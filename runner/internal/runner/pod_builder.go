package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// PodBuilder builds pods using the Builder pattern.
// It provides a fluent API for configuring and creating pods.
// Uses Proto types directly for zero-copy message passing.
type PodBuilder struct {
	runner *Runner

	// Pod command (Proto type)
	cmd *runnerv1.CreatePodCommand

	// Terminal configuration
	rows int
	cols int
}

// NewPodBuilder creates a new pod builder.
func NewPodBuilder(runner *Runner) *PodBuilder {
	return &PodBuilder{
		runner: runner,
		rows:   24,
		cols:   80,
	}
}

// WithCommand sets the create pod command (Proto type).
// This is the primary way to configure the pod.
func (b *PodBuilder) WithCommand(cmd *runnerv1.CreatePodCommand) *PodBuilder {
	b.cmd = cmd
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

	// Merge environment variables
	envVars := b.mergeEnvVars()
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
		OnOutput: nil, // Will be set by caller
		OnExit:   nil, // Will be set by caller
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

	// Create pod
	pod := &Pod{
		ID:          b.cmd.PodKey,
		PodKey:      b.cmd.PodKey,
		AgentType:   "", // Could be extracted from command if needed
		Branch:      branchName,
		SandboxPath: sandboxRoot,
		Terminal:    term,
		StartedAt:   time.Now(),
		Status:      PodStatusInitializing,
	}

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
func (b *PodBuilder) mergeEnvVars() map[string]string {
	result := make(map[string]string)

	// Add config env vars first (lowest priority)
	if b.runner.cfg != nil {
		for k, v := range b.runner.cfg.AgentEnvVars {
			result[k] = v
		}
	}

	// Add command env vars (highest priority)
	if b.cmd != nil {
		for k, v := range b.cmd.EnvVars {
			result[k] = v
		}
	}

	return result
}

// sendProgress sends a pod initialization progress event to the server.
// This is a best-effort operation - errors are logged but not returned.
func (b *PodBuilder) sendProgress(phase string, progress int, message string) {
	if b.cmd == nil || b.cmd.PodKey == "" || b.runner == nil || b.runner.conn == nil {
		return
	}

	if err := b.runner.conn.SendPodInitProgress(b.cmd.PodKey, phase, int32(progress), message); err != nil {
		logger.Pod().Debug("Failed to send init progress", "pod_key", b.cmd.PodKey, "phase", phase, "error", err)
	}
}
