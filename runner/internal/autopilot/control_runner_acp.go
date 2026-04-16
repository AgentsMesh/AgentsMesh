package autopilot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/agents/claude"
)

// AcpControlProcess maintains a long-lived ACP session with the control agent.
// Instead of exec-ing a new process per iteration, it keeps Claude running
// and sends prompts via ACPClient.SendPrompt().
type AcpControlProcess struct {
	client         *acp.ACPClient
	clientConfig   acp.ClientConfig // stored for restart
	promptBuilder  *PromptBuilder
	decisionParser *DecisionParser
	log            *slog.Logger

	// Per-iteration output collection, guarded by mu.
	mu       sync.Mutex
	output   strings.Builder
	iterDone chan struct{}
	started  bool
}

// AcpControlProcessConfig configures the ACP-based control process.
type AcpControlProcessConfig struct {
	Command        string // CLI command (e.g. "claude")
	Args           []string
	WorkDir        string
	Env            []string
	MCPConfigPath  string
	PromptBuilder  *PromptBuilder
	DecisionParser *DecisionParser
	Logger         *slog.Logger
}

// NewAcpControlProcess creates a new ACP-based control process (not started yet).
func NewAcpControlProcess(cfg AcpControlProcessConfig) *AcpControlProcess {
	p := &AcpControlProcess{
		promptBuilder:  cfg.PromptBuilder,
		decisionParser: cfg.DecisionParser,
		log:            cfg.Logger,
	}

	command := cfg.Command
	if command == "" {
		command = DefaultAgent
	}

	// Build launch args for ACP stream-json mode
	args := []string{"--dangerously-skip-permissions"}
	if cfg.MCPConfigPath != "" {
		args = append(args, "--mcp-config", cfg.MCPConfigPath)
	}
	args = append(args, cfg.Args...)

	clientCfg := acp.ClientConfig{
		Command:       command,
		Args:          args,
		WorkDir:       cfg.WorkDir,
		Env:           cfg.Env,
		Logger:        cfg.Logger,
		TransportType: claude.TransportType,
		Callbacks: acp.EventCallbacks{
			OnContentChunk: p.onContentChunk,
			OnStateChange:  p.onStateChange,
			OnThinkingUpdate: func(_ string, update acp.ThinkingUpdate) {
				if p.log != nil {
					p.log.Debug("Control agent thinking", "text_len", len(update.Text))
				}
			},
			OnToolCallUpdate: func(_ string, update acp.ToolCallUpdate) {
				if p.log != nil {
					p.log.Debug("Control agent tool call", "tool", update.ToolName, "status", update.Status)
				}
			},
			OnLog: func(level, message string) {
				if p.log != nil {
					p.log.Debug("Control agent log", "level", level, "message", message)
				}
			},
			OnExit: func(exitCode int) {
				if p.log != nil {
					p.log.Warn("Control agent exited unexpectedly", "exit_code", exitCode)
				}
				// Signal iteration done on unexpected exit so RunControlProcess unblocks.
				p.signalDone()
			},
		},
	}

	p.clientConfig = clientCfg
	p.client = acp.NewClient(clientCfg)

	return p
}

// RunControlProcess sends a prompt and waits for the agent to finish.
// Thread-safety: the caller (AutopilotController) guarantees that only one
// RunControlProcess call is active at a time (enforced by wg + wgMu).
func (p *AcpControlProcess) RunControlProcess(ctx context.Context, iteration int) (*ControlDecision, error) {
	// Start or restart the ACP client if needed.
	if err := p.ensureClientRunning(); err != nil {
		return nil, err
	}

	// If the client is not idle (e.g. previous iteration timed out while agent was
	// still processing), we cannot send a new prompt.
	if state := p.client.State(); state != acp.StateIdle {
		return nil, fmt.Errorf("control agent not idle (state=%s), cannot send prompt", state)
	}

	// Build prompt for this iteration.
	var prompt string
	if iteration <= 1 {
		prompt = p.promptBuilder.BuildPrompt()
	} else {
		prompt = p.promptBuilder.BuildResumePrompt(iteration)
	}

	// Reset output collector for this iteration.
	p.mu.Lock()
	p.output.Reset()
	p.iterDone = make(chan struct{})
	p.mu.Unlock()

	// Send prompt.
	if err := p.client.SendPrompt(prompt); err != nil {
		return nil, fmt.Errorf("send prompt to control agent: %w", err)
	}

	// Wait for completion (OnStateChange → idle) or timeout.
	select {
	case <-p.iterDone:
		// Agent finished (idle state).
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Collect output and parse decision.
	p.mu.Lock()
	outputText := p.output.String()
	p.mu.Unlock()

	if outputText == "" {
		return nil, fmt.Errorf("control agent produced no output")
	}

	decision := p.decisionParser.ParseDecision(outputText)
	return decision, nil
}

// SetSessionID is a no-op for ACP mode (session is managed internally by ACPClient).
func (p *AcpControlProcess) SetSessionID(_ string) {}

// GetSessionID returns the ACP client's session ID.
func (p *AcpControlProcess) GetSessionID() string {
	if p.client != nil {
		return p.client.SessionID()
	}
	return ""
}

// Stop shuts down the ACP client.
func (p *AcpControlProcess) Stop() {
	if p.started && p.client != nil {
		p.client.Stop()
		p.started = false
	}
}

// ensureClientRunning starts or restarts the ACP client if it's not running.
// Handles initial startup and recovery after crashes.
func (p *AcpControlProcess) ensureClientRunning() error {
	if p.started {
		// Check if the client is still alive (not stopped/crashed).
		state := p.client.State()
		if state != acp.StateStopped {
			return nil // Client is running.
		}
		// Client crashed — need to restart.
		p.log.Warn("ACP control agent stopped unexpectedly, restarting")
		p.client.Stop() // Clean up old resources.
		p.started = false
		// Recreate the client with same config.
		p.client = p.recreateClient()
	}

	if err := p.client.Start(); err != nil {
		return fmt.Errorf("start ACP control agent: %w", err)
	}
	if err := p.client.NewSession(nil); err != nil {
		p.log.Warn("Failed to create ACP session (non-fatal)", "error", err)
	}
	p.started = true
	return nil
}

// --- Internal helpers ---

// recreateClient creates a fresh ACPClient using the stored config.
// Used for crash recovery — the old client is stopped before calling this.
func (p *AcpControlProcess) recreateClient() *acp.ACPClient {
	return acp.NewClient(p.clientConfig)
}

// onContentChunk accumulates assistant output for decision parsing.
func (p *AcpControlProcess) onContentChunk(_ string, chunk acp.ContentChunk) {
	if chunk.Role != "assistant" {
		return
	}
	p.mu.Lock()
	p.output.WriteString(chunk.Text)
	p.mu.Unlock()
}

// onStateChange detects when the agent returns to idle (iteration complete).
func (p *AcpControlProcess) onStateChange(newState string) {
	if newState == acp.StateIdle {
		p.signalDone()
	}
}

// signalDone closes iterDone channel if not already closed.
func (p *AcpControlProcess) signalDone() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.iterDone != nil {
		select {
		case <-p.iterDone:
			// already closed
		default:
			close(p.iterDone)
		}
	}
}
