package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
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
func (b *PodBuilder) WithTerminalSize(rows, cols int) *PodBuilder {
	if rows > 0 {
		b.rows = rows
	}
	if cols > 0 {
		b.cols = cols
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
	sandboxRoot, workingDir, worktreePath, branchName, err := b.setup(ctx)
	if err != nil {
		return nil, err
	}

	// Resolve template variables in launch args
	resolvedArgs := b.resolveArgs(b.cmd.LaunchArgs, sandboxRoot, workingDir)

	// Merge environment variables
	envVars := b.mergeEnvVars()

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
		ID:           b.cmd.PodKey,
		PodKey:       b.cmd.PodKey,
		AgentType:    "", // Could be extracted from command if needed
		Branch:       branchName,
		WorktreePath: worktreePath,
		Terminal:     term,
		StartedAt:    time.Now(),
		Status:       PodStatusInitializing,
	}

	logger.Pod().Info("Pod built", "pod_key", b.cmd.PodKey, "working_dir", workingDir)

	// Report progress: ready
	b.sendProgress("ready", 100, "Pod is ready")

	return pod, nil
}

// setup sets up the sandbox and working directory.
// Returns (sandboxRoot, workingDir, worktreePath, branchName, error).
func (b *PodBuilder) setup(ctx context.Context) (string, string, string, string, error) {
	// 1. Create sandbox root directory
	b.sendProgress("preparing", 10, "Creating sandbox directory...")
	sandboxRoot := filepath.Join(b.runner.cfg.WorkspaceRoot, "sandboxes", b.cmd.PodKey)
	if err := os.MkdirAll(sandboxRoot, 0755); err != nil {
		return "", "", "", "", &client.PodError{
			Code:    client.ErrCodeSandboxCreate,
			Message: fmt.Sprintf("failed to create sandbox directory: %v", err),
		}
	}

	cfg := b.cmd.SandboxConfig

	// 2. Setup working directory based on SandboxConfig
	b.sendProgress("preparing", 20, "Setting up working directory...")

	var workingDir, worktreePath, branchName string
	var err error

	if cfg != nil && cfg.RepositoryUrl != "" {
		// Has repository - create worktree
		worktreePath, branchName, err = b.setupGitWorktree(ctx, sandboxRoot, cfg)
		if err != nil {
			os.RemoveAll(sandboxRoot)
			return "", "", "", "", err
		}
		workingDir = worktreePath

		// Run preparation script if configured
		if cfg.PreparationScript != "" {
			if err := b.runPreparationScript(ctx, cfg, worktreePath, branchName); err != nil {
				os.RemoveAll(sandboxRoot)
				return "", "", "", "", err
			}
		}
	} else if cfg != nil && cfg.LocalPath != "" {
		// Local path mode
		if _, err := os.Stat(cfg.LocalPath); os.IsNotExist(err) {
			os.RemoveAll(sandboxRoot)
			return "", "", "", "", &client.PodError{
				Code:    client.ErrCodeWorkDirNotExist,
				Message: fmt.Sprintf("local path does not exist: %s", cfg.LocalPath),
				Details: map[string]string{"path": cfg.LocalPath},
			}
		}
		workingDir = cfg.LocalPath
	} else {
		// No repository - create empty sandbox workspace
		workingDir = filepath.Join(sandboxRoot, "workspace")
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			os.RemoveAll(sandboxRoot)
			return "", "", "", "", &client.PodError{
				Code:    client.ErrCodeSandboxCreate,
				Message: fmt.Sprintf("failed to create temp workspace: %v", err),
			}
		}
	}

	// 3. Create files from FilesToCreate
	if len(b.cmd.FilesToCreate) > 0 {
		b.sendProgress("preparing", 70, "Creating files...")
	}
	if err := b.createFiles(sandboxRoot, workingDir); err != nil {
		os.RemoveAll(sandboxRoot)
		return "", "", "", "", err
	}

	return sandboxRoot, workingDir, worktreePath, branchName, nil
}

// setupGitWorktree creates a git worktree for the pod.
func (b *PodBuilder) setupGitWorktree(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (string, string, error) {
	if cfg.RepositoryUrl == "" {
		return "", "", &client.PodError{
			Code:    client.ErrCodeGitClone,
			Message: "repository_url is required for worktree creation",
		}
	}

	// Use workspace manager if available
	if b.runner.workspace == nil {
		return "", "", &client.PodError{
			Code:    client.ErrCodeGitWorktree,
			Message: "workspace manager not available for git operations",
		}
	}

	// Report cloning progress
	b.sendProgress("cloning", 30, "Cloning repository...")

	// Build worktree options based on credential type
	opts := []workspace.WorktreeOption{}

	switch cfg.CredentialType {
	case "runner_local":
		// Use Runner's local git configuration, no credentials needed
		logger.Pod().Debug("Using runner local git config", "pod_key", b.cmd.PodKey)
	case "oauth", "pat":
		// HTTPS + token authentication
		if cfg.GitToken != "" {
			opts = append(opts, workspace.WithGitToken(cfg.GitToken))
		}
	case "ssh_key":
		// SSH private key authentication
		if cfg.SshPrivateKey != "" {
			// Write SSH private key to temporary file in sandbox
			keyFile := filepath.Join(sandboxRoot, ".ssh_key")
			if err := os.WriteFile(keyFile, []byte(cfg.SshPrivateKey), 0600); err != nil {
				return "", "", &client.PodError{
					Code:    client.ErrCodeFileCreate,
					Message: fmt.Sprintf("failed to write SSH key: %v", err),
				}
			}
			opts = append(opts, workspace.WithSSHKeyPath(keyFile))
			logger.Pod().Debug("SSH key written to sandbox", "pod_key", b.cmd.PodKey, "key_file", keyFile)
		}
	default:
		// Unknown type - fallback to runner_local behavior
		if cfg.CredentialType != "" {
			logger.Pod().Warn("Unknown credential type, using runner local",
				"credential_type", cfg.CredentialType, "pod_key", b.cmd.PodKey)
		}
	}

	worktreePath, err := b.runner.workspace.CreateWorktreeWithOptions(
		ctx,
		cfg.RepositoryUrl,
		cfg.SourceBranch,
		b.cmd.PodKey,
		opts...,
	)
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		errCode := client.ErrCodeGitWorktree
		if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "Permission denied") {
			errCode = client.ErrCodeGitAuth
		} else if strings.Contains(errMsg, "clone") {
			errCode = client.ErrCodeGitClone
		}
		return "", "", &client.PodError{
			Code:    errCode,
			Message: fmt.Sprintf("failed to create worktree: %v", err),
			Details: map[string]string{
				"repository": cfg.RepositoryUrl,
				"branch":     cfg.SourceBranch,
			},
		}
	}

	// Report progress after successful clone
	b.sendProgress("cloning", 60, "Repository cloned successfully")

	branchName := cfg.SourceBranch
	if branchName == "" {
		branchName = "main"
	}
	return worktreePath, branchName, nil
}

// runPreparationScript executes the preparation script in the worktree.
func (b *PodBuilder) runPreparationScript(ctx context.Context, cfg *runnerv1.SandboxConfig, worktreePath, branchName string) error {
	timeout := int(cfg.PreparationTimeout)
	if timeout <= 0 {
		timeout = 300 // Default 5 minutes
	}

	b.sendProgress("preparing", 65, "Running preparation script...")

	preparer := workspace.NewPreparerFromScript(cfg.PreparationScript, timeout)
	if preparer == nil {
		return nil
	}

	prepCtx := &workspace.PreparationContext{
		PodID:            b.cmd.PodKey,
		TicketIdentifier: cfg.TicketId,
		BranchName:       branchName,
		WorkingDir:       worktreePath,
		WorktreeDir:      worktreePath,
	}

	if err := preparer.Prepare(ctx, prepCtx); err != nil {
		return &client.PodError{
			Code:    client.ErrCodePrepareScript,
			Message: fmt.Sprintf("preparation script failed: %v", err),
		}
	}

	b.sendProgress("preparing", 75, "Preparation script completed")
	return nil
}

// createFiles creates files from the FilesToCreate list.
func (b *PodBuilder) createFiles(sandboxRoot, workDir string) error {
	for _, f := range b.cmd.FilesToCreate {
		// Resolve path template
		path := b.resolvePath(f.Path, sandboxRoot, workDir)

		if f.IsDirectory {
			if err := os.MkdirAll(path, 0755); err != nil {
				return &client.PodError{
					Code:    client.ErrCodeFileCreate,
					Message: fmt.Sprintf("failed to create directory: %v", err),
					Details: map[string]string{"path": path},
				}
			}
			continue
		}

		// Ensure parent directory exists
		parentDir := filepath.Dir(path)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to create parent directory: %v", err),
				Details: map[string]string{"path": parentDir},
			}
		}

		// Determine file mode
		mode := os.FileMode(0644)
		if f.Mode != 0 {
			mode = os.FileMode(f.Mode)
		}

		// Write file
		if err := os.WriteFile(path, []byte(f.Content), mode); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to write file: %v", err),
				Details: map[string]string{"path": path},
			}
		}

		logger.Pod().Debug("Created file", "path", path, "mode", fmt.Sprintf("%o", mode))
	}

	return nil
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
