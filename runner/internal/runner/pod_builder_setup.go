package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// setup sets up the sandbox and working directory.
// Returns (sandboxRoot, workingDir, branchName, error).
func (b *PodBuilder) setup(ctx context.Context) (string, string, string, error) {
	// 1. Create sandbox root directory
	b.sendProgress("preparing", 10, "Creating sandbox directory...")
	sandboxRoot := filepath.Join(b.runner.cfg.WorkspaceRoot, "sandboxes", b.cmd.PodKey)
	if err := os.MkdirAll(sandboxRoot, 0755); err != nil {
		return "", "", "", &client.PodError{
			Code:    client.ErrCodeSandboxCreate,
			Message: fmt.Sprintf("failed to create sandbox directory: %v", err),
		}
	}
	logger.Pod().Debug("Sandbox root created", "pod_key", b.cmd.PodKey, "path", sandboxRoot)

	cfg := b.cmd.SandboxConfig

	// 2. Setup working directory based on SandboxConfig
	b.sendProgress("preparing", 20, "Setting up working directory...")

	// Determine setup mode
	var setupMode string
	if cfg != nil && cfg.RepositoryUrl != "" {
		setupMode = "git_worktree"
	} else if cfg != nil && cfg.LocalPath != "" {
		setupMode = "local_path"
	} else {
		setupMode = "empty_sandbox"
	}
	logger.Pod().Debug("Working directory setup mode", "pod_key", b.cmd.PodKey, "mode", setupMode)

	var workingDir, branchName string
	var err error

	if cfg != nil && cfg.RepositoryUrl != "" {
		// Has repository - create git worktree as workspace
		workingDir, branchName, err = b.setupGitWorktree(ctx, sandboxRoot, cfg)
		if err != nil {
			os.RemoveAll(sandboxRoot)
			return "", "", "", err
		}

		// Run preparation script if configured
		if cfg.PreparationScript != "" {
			if err := b.runPreparationScript(ctx, cfg, workingDir, branchName); err != nil {
				os.RemoveAll(sandboxRoot)
				return "", "", "", err
			}
		}
	} else if cfg != nil && cfg.LocalPath != "" {
		// Local path mode (Resume mode) - use existing sandbox directly
		if _, err := os.Stat(cfg.LocalPath); os.IsNotExist(err) {
			os.RemoveAll(sandboxRoot)
			return "", "", "", &client.PodError{
				Code:    client.ErrCodeWorkDirNotExist,
				Message: fmt.Sprintf("local path does not exist: %s", cfg.LocalPath),
				Details: map[string]string{"path": cfg.LocalPath},
			}
		}
		// For resume mode, the working directory is the workspace inside the existing sandbox
		workingDir = filepath.Join(cfg.LocalPath, "workspace")
		if _, err := os.Stat(workingDir); os.IsNotExist(err) {
			// Fallback to local path itself if workspace subdirectory doesn't exist
			workingDir = cfg.LocalPath
		}
	} else {
		// No repository - create empty sandbox workspace
		workingDir = filepath.Join(sandboxRoot, "workspace")
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			os.RemoveAll(sandboxRoot)
			return "", "", "", &client.PodError{
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
		return "", "", "", err
	}

	logger.Pod().Info("Sandbox setup completed",
		"pod_key", b.cmd.PodKey,
		"sandbox_root", sandboxRoot,
		"working_dir", workingDir,
		"branch", branchName)

	return sandboxRoot, workingDir, branchName, nil
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
	logger.Pod().Debug("Setting up git credentials", "pod_key", b.cmd.PodKey, "credential_type", cfg.CredentialType)

	switch cfg.CredentialType {
	case "runner_local":
		// Use Runner's local git configuration, no credentials needed
		logger.Pod().Debug("Using runner local git config", "pod_key", b.cmd.PodKey)
	case "oauth", "pat":
		// HTTPS + token authentication
		logger.Pod().Debug("Using token authentication", "pod_key", b.cmd.PodKey, "type", cfg.CredentialType)
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

	// Create git worktree inside sandbox directory: sandboxes/{podKey}/workspace
	workspaceTarget := filepath.Join(sandboxRoot, "workspace")
	workspacePath, err := b.runner.workspace.CreateWorktreeWithOptions(
		ctx,
		cfg.RepositoryUrl,
		cfg.SourceBranch,
		workspaceTarget,
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
			Message: fmt.Sprintf("failed to create workspace: %v", err),
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

	logger.Pod().Info("Git worktree created",
		"pod_key", b.cmd.PodKey,
		"workspace", workspacePath,
		"branch", branchName)

	return workspacePath, branchName, nil
}

// runPreparationScript executes the preparation script in the workspace.
func (b *PodBuilder) runPreparationScript(ctx context.Context, cfg *runnerv1.SandboxConfig, workspacePath, branchName string) error {
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
		WorkspaceDir:     workspacePath,
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
