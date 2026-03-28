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
	if b.cmd.PodfileSource == "" {
		return nil, &client.PodError{
			Code:    client.ErrCodePodfileEval,
			Message: "podfile source is required",
		}
	}

	launchCommand := b.cmd.LaunchCommand
	logger.Pod().Info("Building pod", "pod_key", b.cmd.PodKey, "command", launchCommand,
		"interaction_mode", b.cmd.GetInteractionMode())

	b.sendProgress("pending", 0, "Initializing pod...")

	sandboxRoot, workingDir, branchName, err := b.setup(ctx)
	if err != nil {
		return nil, err
	}

	pfResult, err := ExecutePodFile(b.cmd, sandboxRoot, workingDir)
	if err != nil {
		return nil, &client.PodError{
			Code:    client.ErrCodePodfileEval,
			Message: fmt.Sprintf("podfile eval failed: %v", err),
		}
	}
	launchCommand = pfResult.LaunchCommand
	resolvedArgs := pfResult.LaunchArgs

	if err := b.createFilesFromProto(pfResult.FilesToCreate, sandboxRoot, workingDir); err != nil {
		return nil, err
	}

	envVars := b.mergeEnvVars(sandboxRoot)
	for k, v := range pfResult.EnvVars {
		envVars[k] = v
	}

	logger.Pod().Debug("Resolved launch args", "pod_key", b.cmd.PodKey, "args", resolvedArgs)
	logger.Pod().Debug("Merged environment variables", "pod_key", b.cmd.PodKey, "count", len(envVars))

	if b.cmd.GetInteractionMode() == "acp" {
		return b.buildACPPod(ctx, sandboxRoot, workingDir, branchName, resolvedArgs, envVars, launchCommand)
	}
	return b.buildPTYPod(ctx, sandboxRoot, workingDir, branchName, resolvedArgs, envVars, launchCommand)
}

// buildPTYPod creates a pod with PTY terminal interaction.
func (b *PodBuilder) buildPTYPod(ctx context.Context, sandboxRoot, workingDir, branchName string, resolvedArgs []string, envVars map[string]string, launchCommand string) (*Pod, error) {
	b.sendProgress("starting_pty", 80, "Starting terminal...")

	// Build PTY factory for Pod Daemon mode (session persistence across restarts)
	var ptyFactory terminal.PTYFactory
	if b.deps.PodDaemonManager != nil && sandboxRoot != "" {
		mgr := b.deps.PodDaemonManager
		opts := poddaemon.CreateOpts{
			PodKey:         b.cmd.PodKey,
			Agent:          launchCommand,
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

	agg := aggregator.NewSmartAggregator(nil, nil, aggregator.WithFullRedrawThrottling())

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
		Terminal:        term,
		VirtualTerminal: virtualTerm,
		Aggregator:      agg,
		PTYLogger:       ptyLogger,
		StartedAt:       time.Now(),
		Status:          PodStatusInitializing,
	}

	ptyIO := NewPTYPodIO(term, virtualTerm, pod)
	ptyIO.SetAggregator(agg)
	if ptyLogger != nil {
		ptyIO.SetPTYLogger(ptyLogger)
	}
	pod.IO = ptyIO
	pod.Relay = NewPTYPodRelay(b.cmd.PodKey, pod.IO, virtualTerm, term, agg)
	term.SetOutputHandler(pod.CreateOutputHandler())

	logger.Pod().Info("Pod built (PTY)", "pod_key", b.cmd.PodKey, "working_dir", workingDir, "cols", b.cols, "rows", b.rows)
	b.sendProgress("ready", 100, "Pod is ready")

	return pod, nil
}

func (b *PodBuilder) resolvePath(pathTemplate, sandboxRoot, workDir string) string {
	path := pathTemplate
	path = strings.ReplaceAll(path, "{{.sandbox.root_path}}", sandboxRoot)
	path = strings.ReplaceAll(path, "{{.sandbox.work_dir}}", workDir)
	return path
}

// mergeEnvVars merges environment variables: resolved PATH < config env < command env.
func (b *PodBuilder) mergeEnvVars(sandboxRoot string) map[string]string {
	result := make(map[string]string)

	// Inject resolved login shell PATH so PTY processes can find tools
	// even when runner runs as a systemd/launchd service with minimal PATH.
	if b.deps.Config != nil && b.deps.Config.ResolvedPATH != "" {
		result["PATH"] = b.deps.Config.ResolvedPATH
	}

	if b.deps.Config != nil {
		for k, v := range b.deps.Config.AgentEnvVars {
			result[k] = v
		}
	}

	if b.cmd != nil {
		for k, v := range b.cmd.EnvVars {
			result[k] = v
		}
	}

	return result
}

// sendProgress sends a pod initialization progress event (best-effort).
func (b *PodBuilder) sendProgress(phase string, progress int, message string) {
	if b.cmd == nil || b.cmd.PodKey == "" || b.deps.ProgressSender == nil {
		return
	}

	if err := b.deps.ProgressSender.SendPodInitProgress(b.cmd.PodKey, phase, int32(progress), message); err != nil {
		logger.Pod().Debug("Failed to send init progress", "pod_key", b.cmd.PodKey, "phase", phase, "error", err)
	}
}
