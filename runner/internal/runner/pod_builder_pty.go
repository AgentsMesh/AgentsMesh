package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// buildPTYPod creates a pod with PTY terminal interaction.
func (b *PodBuilder) buildPTYPod(ctx context.Context, sandboxRoot, workingDir, branchName string, resolvedArgs []string, envVars map[string]string, launchCommand string) (*Pod, error) {
	b.sendProgress("starting_pty", 80, "Starting terminal...")

	// capturedEnv holds the full merged environment (os.Environ + AgentFile ENV)
	// as built by terminal.New. Replicated here for perpetual pod restart.
	capturedEnv := buildMergedEnv(envVars)

	// Inject W3C trace context into envVars map so terminal.New() propagates it
	// to the child process via both the ptyFactory (daemon mode) and direct PTY.
	injectTraceparent(ctx, envVars)
	if tp, ok := envVars["TRACEPARENT"]; ok {
		capturedEnv = append(capturedEnv, "TRACEPARENT="+tp)
	}

	// Build PTY factory for Pod Daemon mode (session persistence across restarts)
	var ptyFactory terminal.PTYFactory
	if b.deps.PodDaemonManager != nil && sandboxRoot != "" {
		mgr := b.deps.PodDaemonManager
		opts := poddaemon.CreateOpts{
			PodKey:         b.cmd.PodKey,
			Agent:          launchCommand,
			SandboxPath:    sandboxRoot,
			WorkDir:        workingDir,
			RepositoryURL:  b.cmd.GetSandboxConfig().GetHttpCloneUrl(),
			Branch:         branchName,
			TicketSlug:     b.cmd.GetSandboxConfig().GetTicketSlug(),
			VTHistoryLimit: b.vtHistoryLimit,
			Perpetual:      b.cmd.Perpetual,
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

	term, err := terminal.New(terminal.Options{
		Command:    launchCommand,
		Args:       resolvedArgs,
		WorkDir:    workingDir,
		Env:        envVars,
		Rows:       b.rows,
		Cols:       b.cols,
		Label:      b.cmd.PodKey,
		PTYFactory: ptyFactory,
	})
	if err != nil {
		if sandboxRoot != "" {
			os.RemoveAll(sandboxRoot)
		}
		return nil, &client.PodError{
			Code:    client.ErrCodeCommandStart,
			Message: fmt.Sprintf("failed to create terminal: %v", err),
		}
	}

	virtualTerm := vt.NewVirtualTerminal(b.cols, b.rows, b.vtHistoryLimit)
	if b.oscHandler != nil {
		virtualTerm.SetOSCHandler(b.oscHandler)
	}

	// Create SmartAggregator for adaptive frame rate output
	agg := aggregator.NewSmartAggregator(nil,
		aggregator.WithFullRedrawThrottling(),
	)

	var ptyLogger *aggregator.PTYLogger
	if b.enablePTYLogging && b.ptyLogDir != "" {
		var logErr error
		ptyLogger, logErr = aggregator.NewPTYLogger(b.ptyLogDir, b.cmd.PodKey)
		if logErr != nil {
			logger.Pod().WarnContext(ctx, "Failed to create PTY logger", "pod_key", b.cmd.PodKey, "error", logErr)
		} else {
			agg.SetPTYLogger(ptyLogger)
			logger.Pod().InfoContext(ctx, "PTY logging enabled for pod", "pod_key", b.cmd.PodKey, "log_dir", ptyLogger.LogDir())
		}
	}

	pod := &Pod{
		ID:              b.cmd.PodKey,
		PodKey:          b.cmd.PodKey,
		Agent:           launchCommand,
		InteractionMode: InteractionModePTY,
		Branch:          branchName,
		SandboxPath:     sandboxRoot,
		LaunchCommand:   launchCommand,
		LaunchArgs:      resolvedArgs,
		WorkDir:         workingDir,
		LaunchEnv:       capturedEnv,
		Perpetual:       b.cmd.Perpetual,
		StartedAt:       time.Now(),
		Status:          PodStatusInitializing,
	}

	pod.installRuntime(assemblePTYRuntime(
		pod, term, virtualTerm, agg, ptyLogger, b.deps.LocalRelayServer,
	))

	logger.Pod().InfoContext(ctx, "Pod built (PTY)", "pod_key", b.cmd.PodKey, "working_dir", workingDir, "cols", b.cols, "rows", b.rows)
	b.sendProgress("ready", 100, "Pod is ready")

	return pod, nil
}
