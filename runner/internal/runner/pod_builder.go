package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/clipboard"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// PodBuilderDeps defines the dependencies for PodBuilder.
// This decouples PodBuilder from the Runner struct, following DIP.
type PodBuilderDeps struct {
	// Config provides workspace configuration.
	Config *config.Config

	// Workspace provides git worktree management.
	// Can be nil if git operations are not needed.
	Workspace workspace.WorkspaceManagerInterface

	// ProgressSender sends pod initialization progress.
	// Can be nil if progress reporting is not needed.
	ProgressSender client.ProgressSender

	// Clipboard provides clipboard backend for image paste support.
	// Can be nil; if nil, clipboard support is disabled.
	Clipboard clipboard.Backend
}

// PodBuilder builds pods using the Builder pattern.
// It provides a fluent API for configuring and creating pods.
// Uses Proto types directly for zero-copy message passing.
type PodBuilder struct {
	deps PodBuilderDeps

	// Pod command (Proto type)
	cmd *runnerv1.CreatePodCommand

	// Terminal configuration
	rows int
	cols int

	// VirtualTerminal configuration
	vtHistoryLimit int

	// PTY logging configuration
	enablePTYLogging bool
	ptyLogDir        string

	// OSC handler (called when OSC sequences are received)
	oscHandler vt.OSCHandler

	// Setup strategies (Strategy Pattern for OCP compliance)
	// Strategies are tried in order; first matching strategy is used.
	setupStrategies []SetupStrategy
}

// NewPodBuilder creates a new pod builder with explicit dependencies.
// This is the preferred constructor for new code.
func NewPodBuilder(deps PodBuilderDeps) *PodBuilder {
	b := &PodBuilder{
		deps:           deps,
		rows:           24,
		cols:           80,
		vtHistoryLimit: 100, // Default scrollback history
	}
	// Register default setup strategies
	b.setupStrategies = DefaultSetupStrategies(b)
	return b
}

// NewPodBuilderFromRunner creates a new pod builder from a Runner instance.
// This maintains backward compatibility with existing code.
func NewPodBuilderFromRunner(runner *Runner) *PodBuilder {
	deps := PodBuilderDeps{
		Config:         runner.cfg,
		ProgressSender: runner.conn,
		Clipboard:      runner.clipboard,
	}
	// Explicitly set Workspace only if not nil to avoid interface nil comparison issues
	if runner.workspace != nil {
		deps.Workspace = runner.workspace
	}
	return NewPodBuilder(deps)
}

// WithCommand sets the create pod command (Proto type).
// This is the primary way to configure the pod.
func (b *PodBuilder) WithCommand(cmd *runnerv1.CreatePodCommand) *PodBuilder {
	b.cmd = cmd
	return b
}

// WithSetupStrategies allows customizing the setup strategies.
// This is useful for testing or extending with custom strategies.
// Strategies are tried in order; first matching strategy is used.
func (b *PodBuilder) WithSetupStrategies(strategies []SetupStrategy) *PodBuilder {
	b.setupStrategies = strategies
	return b
}

// WithTerminalSize sets terminal dimensions.
// Parameters follow the standard convention: cols (width) first, then rows (height).
// This matches xterm.js, ANSI standards, and most terminal libraries.
func (b *PodBuilder) WithTerminalSize(cols, rows int) *PodBuilder {
	if cols > 0 {
		b.cols = cols
	}
	if rows > 0 {
		b.rows = rows
	}
	return b
}

// WithPTYLogging enables PTY logging to the specified directory.
func (b *PodBuilder) WithPTYLogging(logDir string) *PodBuilder {
	b.enablePTYLogging = true
	b.ptyLogDir = logDir
	return b
}

// WithOSCHandler sets the handler for OSC (Operating System Command) sequences.
func (b *PodBuilder) WithOSCHandler(handler vt.OSCHandler) *PodBuilder {
	b.oscHandler = handler
	return b
}

// WithVirtualTerminalHistoryLimit sets the scrollback history limit.
func (b *PodBuilder) WithVirtualTerminalHistoryLimit(limit int) *PodBuilder {
	if limit > 0 {
		b.vtHistoryLimit = limit
	}
	return b
}

// Build creates the pod.
func (b *PodBuilder) Build(ctx context.Context) (*Pod, error) {
	if b.cmd == nil {
		return nil, fmt.Errorf("command is required")
	}
	if b.cmd.PodKey == "" {
		return nil, fmt.Errorf("pod key is required")
	}
	if b.cmd.LaunchCommand == "" {
		return nil, fmt.Errorf("launch command is required")
	}

	logger.Pod().Info("Building pod", "pod_key", b.cmd.PodKey, "command", b.cmd.LaunchCommand)

	// Report initial progress
	b.sendProgress("pending", 0, "Initializing pod...")

	// Setup sandbox and working directory
	sandboxRoot, workingDir, branchName, err := b.setup(ctx)
	if err != nil {
		return nil, err
	}

	// Setup clipboard backend for image paste support
	if sandboxRoot != "" && b.deps.Clipboard != nil {
		if err := b.deps.Clipboard.Setup(sandboxRoot); err != nil {
			logger.Pod().Warn("Failed to setup clipboard backend", "backend", b.deps.Clipboard.Name(), "error", err)
			// Non-fatal: image paste won't work but pod still functions
		}
	}

	// Resolve template variables in launch args
	resolvedArgs := b.resolveArgs(b.cmd.LaunchArgs, sandboxRoot, workingDir)
	logger.Pod().Debug("Resolved launch args", "pod_key", b.cmd.PodKey, "args", resolvedArgs)

	// Merge environment variables
	envVars := b.mergeEnvVars(sandboxRoot)
	logger.Pod().Debug("Merged environment variables", "pod_key", b.cmd.PodKey, "count", len(envVars))

	// Report progress: starting PTY
	b.sendProgress("starting_pty", 80, "Starting terminal...")

	// Create terminal
	term, err := terminal.New(terminal.Options{
		Command:  b.cmd.LaunchCommand,
		Args:     resolvedArgs,
		WorkDir:  workingDir,
		Env:      envVars,
		Rows:     b.rows,
		Cols:     b.cols,
		OnOutput: nil, // Will be wired up after all components are created
		OnExit:   nil, // Will be set by caller (MessageHandler)
	})
	if err != nil {
		// Cleanup sandbox on failure
		if sandboxRoot != "" {
			os.RemoveAll(sandboxRoot)
		}
		return nil, &client.PodError{
			Code:    client.ErrCodeCommandStart,
			Message: fmt.Sprintf("failed to create terminal: %v", err),
		}
	}

	// Create VirtualTerminal for terminal state management and snapshots
	virtualTerm := vt.NewVirtualTerminal(b.cols, b.rows, b.vtHistoryLimit)
	if b.oscHandler != nil {
		virtualTerm.SetOSCHandler(b.oscHandler)
	}

	// Create SmartAggregator for adaptive frame rate output
	agg := aggregator.NewSmartAggregator(nil, nil,
		aggregator.WithFullRedrawThrottling(),
	)

	// Set up PTY logging if enabled
	var ptyLogger *aggregator.PTYLogger
	if b.enablePTYLogging && b.ptyLogDir != "" {
		var logErr error
		ptyLogger, logErr = aggregator.NewPTYLogger(b.ptyLogDir, b.cmd.PodKey)
		if logErr != nil {
			logger.Pod().Warn("Failed to create PTY logger", "pod_key", b.cmd.PodKey, "error", logErr)
		} else {
			agg.SetPTYLogger(ptyLogger)
			logger.Pod().Info("PTY logging enabled for pod", "pod_key", b.cmd.PodKey, "log_dir", ptyLogger.LogDir())
		}
	}

	// Create pod with all components
	pod := &Pod{
		ID:              b.cmd.PodKey,
		PodKey:          b.cmd.PodKey,
		AgentType:       "", // Could be extracted from command if needed
		Branch:          branchName,
		SandboxPath:     sandboxRoot,
		Clipboard:       b.deps.Clipboard,
		Terminal:        term,
		VirtualTerminal: virtualTerm,
		Aggregator:      agg,
		PTYLogger:       ptyLogger,
		StartedAt:       time.Now(),
		Status:          PodStatusInitializing,
	}

	// Wire up output handler (integrates VirtualTerminal, StateDetector, and Aggregator)
	podKey := b.cmd.PodKey
	term.SetOutputHandler(func(data []byte) {
		defer func() {
			if r := recover(); r != nil {
				logger.Terminal().Error("PANIC in OutputHandler recovered",
					"pod_key", podKey,
					"panic", fmt.Sprintf("%v", r),
					"data_len", len(data))
			}
		}()

		var screenLines []string
		if virtualTerm != nil {
			screenLines = virtualTerm.Feed(data)
		}
		go pod.NotifyStateDetectorWithScreen(len(data), screenLines)
		agg.Write(data)
	})

	logger.Pod().Info("Pod built", "pod_key", b.cmd.PodKey, "working_dir", workingDir, "cols", b.cols, "rows", b.rows)

	// Report progress: ready
	b.sendProgress("ready", 100, "Pod is ready")

	return pod, nil
}

// resolvePath resolves path template variables.
func (b *PodBuilder) resolvePath(pathTemplate, sandboxRoot, workDir string) string {
	path := pathTemplate
	path = strings.ReplaceAll(path, "{{.sandbox.root_path}}", sandboxRoot)
	path = strings.ReplaceAll(path, "{{.sandbox.work_dir}}", workDir)
	return path
}

// resolveArgs resolves template variables in command line arguments.
func (b *PodBuilder) resolveArgs(args []string, sandboxRoot, workDir string) []string {
	resolved := make([]string, len(args))
	for i, arg := range args {
		resolved[i] = b.resolvePath(arg, sandboxRoot, workDir)
	}
	return resolved
}

// mergeEnvVars merges all environment variable sources.
func (b *PodBuilder) mergeEnvVars(sandboxRoot string) map[string]string {
	result := make(map[string]string)

	// Inject resolved login shell PATH as lowest-priority base.
	// This ensures PTY processes can find tools (claude, aider, etc.)
	// even when the runner runs as a systemd/launchd service with minimal PATH.
	if b.deps.Config != nil && b.deps.Config.ResolvedPATH != "" {
		result["PATH"] = b.deps.Config.ResolvedPATH
	}

	// Add config env vars (overrides resolved PATH if user sets PATH in config)
	if b.deps.Config != nil {
		for k, v := range b.deps.Config.AgentEnvVars {
			result[k] = v
		}
	}

	// Add command env vars (highest priority)
	if b.cmd != nil {
		for k, v := range b.cmd.EnvVars {
			result[k] = v
		}
	}

	// Apply clipboard backend env overrides (e.g., PATH prepend for shim)
	if sandboxRoot != "" && b.deps.Clipboard != nil {
		overrides := b.deps.Clipboard.EnvOverrides(sandboxRoot)
		for k, v := range overrides {
			if k == "PATH" {
				// v is a directory to prepend to PATH
				basePath := result["PATH"]
				if basePath == "" {
					basePath = os.Getenv("PATH")
				}
				if basePath == "" {
					result["PATH"] = v
				} else {
					result["PATH"] = v + string(os.PathListSeparator) + basePath
				}
				continue
			}
			result[k] = v
		}
	}

	return result
}

// sendProgress sends a pod initialization progress event to the server.
// This is a best-effort operation - errors are logged but not returned.
func (b *PodBuilder) sendProgress(phase string, progress int, message string) {
	if b.cmd == nil || b.cmd.PodKey == "" || b.deps.ProgressSender == nil {
		return
	}

	if err := b.deps.ProgressSender.SendPodInitProgress(b.cmd.PodKey, phase, int32(progress), message); err != nil {
		logger.Pod().Debug("Failed to send init progress", "pod_key", b.cmd.PodKey, "phase", phase, "error", err)
	}
}
