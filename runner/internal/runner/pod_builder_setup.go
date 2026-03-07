package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/cache"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/fsutil"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// setup sets up the sandbox and working directory.
// Returns (sandboxRoot, workingDir, branchName, error).
// Uses Strategy Pattern to select the appropriate setup strategy based on SandboxConfig.
func (b *PodBuilder) setup(ctx context.Context) (string, string, string, error) {
	// 1. Create sandbox root directory
	b.sendProgress("preparing", 10, "Creating sandbox directory...")
	sandboxRoot := filepath.Join(b.deps.Config.WorkspaceRoot, "sandboxes", b.cmd.PodKey)
	if err := os.MkdirAll(sandboxRoot, 0755); err != nil {
		return "", "", "", &client.PodError{
			Code:    client.ErrCodeSandboxCreate,
			Message: fmt.Sprintf("failed to create sandbox directory: %v", err),
		}
	}
	logger.Pod().Debug("Sandbox root created", "pod_key", b.cmd.PodKey, "path", sandboxRoot)

	cfg := b.cmd.SandboxConfig

	// 2. Select and execute setup strategy
	b.sendProgress("preparing", 20, "Setting up working directory...")

	strategy := b.selectSetupStrategy(cfg)
	logger.Pod().Debug("Working directory setup mode", "pod_key", b.cmd.PodKey, "mode", strategy.Name())

	result, err := strategy.Setup(ctx, sandboxRoot, cfg)
	if err != nil {
		if rmErr := fsutil.RemoveAll(sandboxRoot); rmErr != nil {
			slog.Warn("Failed to clean up sandbox after setup error", "path", sandboxRoot, "error", rmErr)
		}
		return "", "", "", err
	}

	// 3. Create files from FilesToCreate
	if len(b.cmd.FilesToCreate) > 0 {
		b.sendProgress("preparing", 70, "Creating files...")
	}
	if err := b.createFiles(sandboxRoot, result.WorkingDir); err != nil {
		if rmErr := fsutil.RemoveAll(sandboxRoot); rmErr != nil {
			slog.Warn("Failed to clean up sandbox after file creation error", "path", sandboxRoot, "error", rmErr)
		}
		return "", "", "", err
	}

	// Download skill packages
	if err := b.downloadResources(ctx, sandboxRoot, result.WorkingDir); err != nil {
		if rmErr := fsutil.RemoveAll(sandboxRoot); rmErr != nil {
			slog.Warn("Failed to clean up sandbox after download error", "path", sandboxRoot, "error", rmErr)
		}
		return "", "", "", fmt.Errorf("failed to download resources: %w", err)
	}

	logger.Pod().Info("Sandbox setup completed",
		"pod_key", b.cmd.PodKey,
		"sandbox_root", sandboxRoot,
		"working_dir", result.WorkingDir,
		"branch", result.BranchName)

	return sandboxRoot, result.WorkingDir, result.BranchName, nil
}

// selectSetupStrategy selects the appropriate setup strategy based on configuration.
// Strategies are tried in order; first matching strategy is used.
func (b *PodBuilder) selectSetupStrategy(cfg *runnerv1.SandboxConfig) SetupStrategy {
	for _, strategy := range b.setupStrategies {
		if strategy.CanHandle(cfg) {
			return strategy
		}
	}
	// Fallback to empty sandbox (should not reach here if strategies are properly configured)
	return NewEmptySandboxStrategy()
}

// setupGitWorktree creates a git worktree for the pod.
func (b *PodBuilder) setupGitWorktree(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (string, string, error) {
	// Determine repository URL: prefer new fields, fall back to legacy repository_url
	repoURL := cfg.RepositoryUrl
	if cfg.HttpCloneUrl != "" || cfg.SshCloneUrl != "" {
		// New fields are available — use HTTP as primary, probe will select the right one
		if cfg.HttpCloneUrl != "" {
			repoURL = cfg.HttpCloneUrl
		} else {
			repoURL = cfg.SshCloneUrl
		}
	}

	if repoURL == "" {
		return "", "", &client.PodError{
			Code:    client.ErrCodeGitClone,
			Message: "repository_url is required for worktree creation",
		}
	}

	// Use workspace manager if available
	if b.deps.Workspace == nil {
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
			// On Windows, os.FileMode(0600) is not enforced by the filesystem.
			// SSH clients require strict permissions, so use icacls to remove
			// inherited ACLs and grant read-only access to the current user.
			if runtime.GOOS == "windows" {
				username := os.Getenv("USERNAME")
				if username == "" {
					// Fallback for Windows Service or container environments
					// where USERNAME may not be set.
					if u, err := user.Current(); err == nil {
						username = u.Username
					}
				}
				if username != "" {
					if err := exec.Command("icacls", keyFile, "/inheritance:r",
						"/grant:r", username+":R").Run(); err != nil {
						logger.Pod().Warn("Failed to set SSH key ACL (SSH may reject key if permissions are too open)",
							"error", err, "key_file", keyFile)
					}
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

	// Pass new clone URLs for smart probing
	if cfg.HttpCloneUrl != "" {
		opts = append(opts, workspace.WithHttpCloneURL(cfg.HttpCloneUrl))
	}
	if cfg.SshCloneUrl != "" {
		opts = append(opts, workspace.WithSshCloneURL(cfg.SshCloneUrl))
	}

	// Create git worktree inside sandbox directory: sandboxes/{podKey}/workspace
	workspaceTarget := filepath.Join(sandboxRoot, "workspace")
	result, err := b.deps.Workspace.CreateWorktreeWithOptions(
		ctx,
		repoURL,
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

	// WorktreeResult.Branch already falls back to the requested branch
	// when detached HEAD is detected, so no additional fallback is needed.
	branchName := result.Branch

	logger.Pod().Info("Git worktree created",
		"pod_key", b.cmd.PodKey,
		"workspace", result.Path,
		"branch", branchName)

	return result.Path, branchName, nil
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
		PodID:        b.cmd.PodKey,
		TicketSlug:   cfg.GetTicketSlug(),
		BranchName:   branchName,
		WorkspaceDir: workspacePath,
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

// downloadResources downloads skill packages and other resources into the sandbox.
func (b *PodBuilder) downloadResources(ctx context.Context, sandboxRoot, workDir string) error {
	if len(b.cmd.ResourcesToDownload) == 0 {
		return nil
	}

	cacheDir := filepath.Join(b.deps.Config.WorkspaceRoot, "cache", "skills")
	cacheManager, err := cache.NewSkillCacheManager(cacheDir)
	if err != nil {
		return fmt.Errorf("failed to create skill cache manager: %w", err)
	}

	downloader := cache.NewDownloader(cacheManager)
	for _, res := range b.cmd.ResourcesToDownload {
		result, err := downloader.DownloadAndExtract(ctx, res, sandboxRoot, workDir)
		if err != nil {
			return fmt.Errorf("failed to download resource %s: %w", res.Sha, err)
		}
		if result.CacheHit {
			slog.Info("Resource cache hit", "sha", res.Sha)
		} else {
			slog.Info("Resource downloaded", "sha", res.Sha, "bytes", result.BytesRead)
		}
	}
	return nil
}

// createFiles creates files from the FilesToCreate list.
func (b *PodBuilder) createFiles(sandboxRoot, workDir string) error {
	absSandbox, err := filepath.Abs(sandboxRoot)
	if err != nil {
		return &client.PodError{
			Code:    client.ErrCodeFileCreate,
			Message: fmt.Sprintf("failed to resolve sandbox root: %v", err),
		}
	}
	absSandbox = filepath.Clean(absSandbox)

	for _, f := range b.cmd.FilesToCreate {
		// Resolve path template
		path := b.resolvePath(f.Path, sandboxRoot, workDir)

		// Validate resolved path stays within sandbox to prevent path traversal attacks
		absPath, err := filepath.Abs(path)
		if err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to resolve file path: %v", err),
				Details: map[string]string{"path": f.Path},
			}
		}
		if absPath != absSandbox && !strings.HasPrefix(absPath, absSandbox+string(os.PathSeparator)) {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("path %q escapes sandbox root", f.Path),
				Details: map[string]string{"path": f.Path},
			}
		}

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
