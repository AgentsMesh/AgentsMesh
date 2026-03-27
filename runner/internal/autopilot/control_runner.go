// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"context"
	"errors"
	"log/slog"
)

// ControlRunner executes the Control Process (Claude CLI).
type ControlRunner struct {
	workDir       string
	agent         string
	mcpConfigPath string // Path to MCP config file for Control Agent
	log           *slog.Logger

	// Session management
	sessionID string

	// Dependencies
	promptBuilder   *PromptBuilder
	decisionParser  *DecisionParser
	commandExecutor CommandExecutor
}

// ControlRunnerConfig contains configuration for creating a ControlRunner.
type ControlRunnerConfig struct {
	WorkDir         string
	Agent           string
	MCPConfigPath   string // Path to MCP config file for Control Agent
	PromptBuilder   *PromptBuilder
	DecisionParser  *DecisionParser
	CommandExecutor CommandExecutor // Optional: defaults to DefaultCommandExecutor
	Logger          *slog.Logger
}

// NewControlRunner creates a new ControlRunner instance.
func NewControlRunner(cfg ControlRunnerConfig) *ControlRunner {
	agent := cfg.Agent
	if agent == "" {
		agent = DefaultAgent
	}

	decisionParser := cfg.DecisionParser
	if decisionParser == nil {
		decisionParser = NewDecisionParser()
	}

	commandExecutor := cfg.CommandExecutor
	if commandExecutor == nil {
		commandExecutor = NewDefaultCommandExecutor()
	}

	return &ControlRunner{
		workDir:         cfg.WorkDir,
		agent:           agent,
		mcpConfigPath:   cfg.MCPConfigPath,
		log:             cfg.Logger,
		promptBuilder:   cfg.PromptBuilder,
		decisionParser:  decisionParser,
		commandExecutor: commandExecutor,
	}
}

// Stop is a no-op for exec-based control runner (process exits after each iteration).
func (cr *ControlRunner) Stop() {}

// RunControlProcess executes the control agent to make a single decision.
// Handles both initial start and session resume.
func (cr *ControlRunner) RunControlProcess(ctx context.Context, iteration int) (*ControlDecision, error) {
	if cr.sessionID == "" {
		return cr.startControlProcess(ctx, iteration)
	}
	return cr.resumeControlProcess(ctx, iteration)
}

// GetSessionID returns the current session ID.
func (cr *ControlRunner) GetSessionID() string {
	return cr.sessionID
}

// SetSessionID sets the session ID (for testing or restore).
func (cr *ControlRunner) SetSessionID(id string) {
	cr.sessionID = id
}

// startControlProcess starts a new Control process for the first iteration.
// This creates a new session and saves the session_id for future resume.
func (cr *ControlRunner) startControlProcess(ctx context.Context, iteration int) (*ControlDecision, error) {
	prompt := cr.promptBuilder.BuildInitialPrompt()

	args := []string{
		"--dangerously-skip-permissions",
		"-p", prompt,
		"--output-format", "json",
	}

	// Add MCP config if available
	if cr.mcpConfigPath != "" {
		args = append(args, "--mcp-config", cr.mcpConfigPath)
	}

	if cr.log != nil {
		cr.log.Info("Starting control process (first iteration)",
			"agent", cr.agent,
			"work_dir", cr.workDir,
			"iteration", iteration)
	}

	stdout, stderr, err := cr.commandExecutor.Execute(ctx, cr.agent, args, cr.workDir)
	if err != nil {
		if errors.Is(err, errOutputTruncated) {
			if cr.log != nil {
				cr.log.Warn("Control process output truncated",
					"error", err,
					"stdout_len", len(stdout),
					"stderr_len", len(stderr))
			}
		} else {
			if cr.log != nil {
				cr.log.Error("Control process failed",
					"error", err,
					"stderr", string(stderr),
					"stdout", string(stdout))
			}
		}
		return nil, err
	}

	output := string(stdout)
	if cr.log != nil {
		cr.log.Debug("Control process output", "output_length", len(output))
	}

	// Try to extract session_id from JSON output
	if sessionID := ExtractSessionID(output); sessionID != "" {
		cr.sessionID = sessionID
		if cr.log != nil {
			cr.log.Info("Extracted session ID", "session_id", cr.sessionID)
		}
	}

	// Extract result text for logging (from JSON if present)
	resultText := ExtractResultFromJSON(output)
	if resultText != "" {
		if cr.log != nil {
			cr.log.Info("Control process result text", "result", resultText)
		}
	} else {
		// Log raw output if not JSON
		logOutput := output
		if len(logOutput) > LogOutputMaxLength {
			logOutput = logOutput[:LogOutputMaxLength] + "... (truncated)"
		}
		if cr.log != nil {
			cr.log.Info("Control process raw output", "output", logOutput)
		}
	}

	decision := cr.decisionParser.ParseDecision(output)
	if cr.log != nil {
		cr.log.Info("Parsed decision", "type", decision.Type, "summary", decision.Summary)
	}

	return decision, nil
}

// resumeControlProcess resumes an existing Control session.
func (cr *ControlRunner) resumeControlProcess(ctx context.Context, iteration int) (*ControlDecision, error) {
	prompt := cr.promptBuilder.BuildResumePrompt(iteration)

	args := []string{
		"--dangerously-skip-permissions",
		"--resume", cr.sessionID,
		"-p", prompt,
		"--output-format", "json",
	}

	// Add MCP config if available
	if cr.mcpConfigPath != "" {
		args = append(args, "--mcp-config", cr.mcpConfigPath)
	}

	if cr.log != nil {
		cr.log.Info("Resuming control process",
			"agent", cr.agent,
			"work_dir", cr.workDir,
			"iteration", iteration,
			"session_id", cr.sessionID)
	}

	stdout, stderr, err := cr.commandExecutor.Execute(ctx, cr.agent, args, cr.workDir)
	if err != nil {
		if errors.Is(err, errOutputTruncated) {
			if cr.log != nil {
				cr.log.Warn("Control process output truncated",
					"error", err,
					"stdout_len", len(stdout),
					"stderr_len", len(stderr))
			}
		} else {
			if cr.log != nil {
				cr.log.Error("Control process failed",
					"error", err,
					"stderr", string(stderr),
					"stdout", string(stdout))
			}
		}
		return nil, err
	}

	output := string(stdout)
	if cr.log != nil {
		cr.log.Debug("Control process output", "output_length", len(output))
	}

	// Extract result text for logging (from JSON if present)
	resultText := ExtractResultFromJSON(output)
	if resultText != "" {
		if cr.log != nil {
			cr.log.Info("Control process result text (resume)", "result", resultText)
		}
	} else {
		// Log raw output if not JSON
		logOutput := output
		if len(logOutput) > LogOutputMaxLength {
			logOutput = logOutput[:LogOutputMaxLength] + "... (truncated)"
		}
		if cr.log != nil {
			cr.log.Info("Control process raw output (resume)", "output", logOutput)
		}
	}

	decision := cr.decisionParser.ParseDecision(output)
	if cr.log != nil {
		cr.log.Info("Parsed decision (resume)", "type", decision.Type, "summary", decision.Summary)
	}

	return decision, nil
}
