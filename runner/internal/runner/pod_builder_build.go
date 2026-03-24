package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

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

	// Resolve template variables in launch args
	resolvedArgs := b.resolveArgs(b.cmd.LaunchArgs, sandboxRoot, workingDir)
	logger.Pod().Debug("Resolved launch args", "pod_key", b.cmd.PodKey, "args", resolvedArgs)

	// Merge environment variables (with template resolution)
	envVars := b.mergeEnvVars(sandboxRoot, workingDir)
	logger.Pod().Debug("Merged environment variables", "pod_key", b.cmd.PodKey, "count", len(envVars))

	// Report progress: starting PTY
	b.sendProgress("starting_pty", 80, "Starting terminal...")

	// Build PTY factory for Pod Daemon mode (session persistence across restarts)
	var ptyFactory terminal.PTYFactory
	if b.deps.PodDaemonManager != nil && sandboxRoot != "" {
		mgr := b.deps.PodDaemonManager
		opts := poddaemon.CreateOpts{
			PodKey:         b.cmd.PodKey,
			AgentType:      b.cmd.LaunchCommand,
			SandboxPath:    sandboxRoot,
			WorkDir:        workingDir,
			RepositoryURL:  b.cmd.GetSandboxConfig().GetRepositoryUrl(),
			Branch:         branchName,
			TicketSlug:     b.cmd.GetSandboxConfig().GetTicketSlug(),
			VTHistoryLimit: b.vtHistoryLimit,
		}
		ptyFactory = func(command string, args []string, workDir string, env []string, cols, rows int) (terminal.PtyProcess, error) {
			opts.Command = command
			opts.Args = args
			opts.Env = env
			opts.Cols = cols
			opts.Rows = rows
			dpty, _, err := mgr.CreateSession(opts)
			if err != nil {
				return nil, err
			}
			return dpty, nil
		}
	}

	// Create terminal
	term, err := terminal.New(terminal.Options{
		Command:    b.cmd.LaunchCommand,
		Args:       resolvedArgs,
		WorkDir:    workingDir,
		Env:        envVars,
		Rows:       b.rows,
		Cols:       b.cols,
		Label:      b.cmd.PodKey, // For log correlation in PTY diagnostics
		PTYFactory: ptyFactory,
		OnOutput:   nil, // Will be wired up after all components are created
		OnExit:     nil, // Will be set by caller (MessageHandler)
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
		AgentType:       b.cmd.LaunchCommand,
		Branch:          branchName,
		SandboxPath:     sandboxRoot,
		LaunchCommand:   b.cmd.LaunchCommand,
		LaunchArgs:      resolvedArgs,
		WorkDir:         workingDir,
		Terminal:        term,
		VirtualTerminal: virtualTerm,
		Aggregator:      agg,
		PTYLogger:       ptyLogger,
		StartedAt:       time.Now(),
		Status:          PodStatusInitializing,
	}

	// Wire up output handler (shared implementation with circuit breaker + inline recover)
	term.SetOutputHandler(pod.CreateOutputHandler())

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

// mergeEnvVars merges all environment variable sources and resolves template variables.
func (b *PodBuilder) mergeEnvVars(sandboxRoot, workDir string) map[string]string {
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

	// Resolve template variables (e.g., {{.sandbox.root_path}}, {{.sandbox.work_dir}})
	for k, v := range result {
		result[k] = b.resolvePath(v, sandboxRoot, workDir)
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
