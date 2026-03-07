package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// WorktreeResult encapsulates the result of a worktree creation operation.
type WorktreeResult struct {
	// Path is the filesystem path where the worktree was created.
	Path string
	// Branch is the actual git branch name checked out in the worktree.
	Branch string
}

// CreateWorktree creates a git worktree for a repository.
// The worktree is created inside the sandbox directory: sandboxes/{podKey}/workspace
func (m *Manager) CreateWorktree(ctx context.Context, repoURL, branch, podKey string) (*WorktreeResult, error) {
	workspacePath := filepath.Join(m.root, "sandboxes", podKey, "workspace")
	return m.CreateWorktreeWithOptions(ctx, repoURL, branch, workspacePath)
}

// CreateWorktreeWithOptions creates a git worktree with additional options.
// worktreePath is the full path where the worktree should be created.
func (m *Manager) CreateWorktreeWithOptions(ctx context.Context, repoURL, branch, worktreePath string, opts ...WorktreeOption) (*WorktreeResult, error) {
	log := logger.Workspace()
	log.Info("Creating worktree", "repo", repoURL, "branch", branch, "path", worktreePath)

	// Apply options
	options := &WorktreeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// If multiple clone URLs are available, probe to find the accessible one
	if options.HttpCloneURL != "" || options.SshCloneURL != "" {
		httpURL := options.HttpCloneURL
		sshURL := options.SshCloneURL

		probeURL, err := m.probeRepositoryAccess(ctx, httpURL, sshURL, options)
		if err != nil {
			return nil, fmt.Errorf("repository access probe failed: %w", err)
		}
		log.Info("Repository access probe selected URL", "url", probeURL)
		repoURL = probeURL
	}

	// Parse repo name from URL (needed before locking)
	repoName := extractRepoName(repoURL)
	if repoName == "" {
		return nil, fmt.Errorf("invalid repository URL: %s", repoURL)
	}
	log.Debug("Parsed repo name", "name", repoName)

	// Per-repo lock: allows concurrent worktree creation for different repositories
	// while serializing git operations (clone/fetch) on the same repository.
	repoLock := m.getRepoLock(repoName)
	repoLock.Lock()
	defer repoLock.Unlock()

	// Main repository path (bare repo cache, shared across pods)
	mainRepoPath := filepath.Join(m.root, "repos", repoName)

	// Clone or fetch the repository with authentication
	if err := m.ensureRepositoryWithAuth(ctx, repoURL, mainRepoPath, options); err != nil {
		return nil, fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Remove existing worktree if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		if err := m.removeWorktreeInternal(ctx, mainRepoPath, worktreePath); err != nil {
			return nil, fmt.Errorf("failed to remove existing worktree: %w", err)
		}
	}

	// Create worktree parent directory
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create worktree parent dir: %w", err)
	}

	// Fetch the branch
	if branch == "" {
		branch = "main"
	}

	// Fetch from remote
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", branch)
	fetchCmd.Dir = mainRepoPath
	m.setGitAuthEnv(fetchCmd, options)
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		// Try 'master' if 'main' fails
		if branch == "main" {
			branch = "master"
			fetchCmd = exec.CommandContext(ctx, "git", "fetch", "origin", branch)
			fetchCmd.Dir = mainRepoPath
			m.setGitAuthEnv(fetchCmd, options)
			if output, err = fetchCmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("failed to fetch branch: %s, output: %s", err, output)
			}
		} else {
			return nil, fmt.Errorf("failed to fetch branch: %s, output: %s", err, output)
		}
	}

	// Create worktree
	// Use a unique branch name based on parent directory name (sandbox podKey)
	// e.g., /path/sandboxes/pod-123/worktree -> worktree-pod-123
	parentDir := filepath.Base(filepath.Dir(worktreePath))
	worktreeBranch := fmt.Sprintf("worktree-%s", parentDir)

	worktreeCmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", worktreeBranch, worktreePath, fmt.Sprintf("origin/%s", branch))
	worktreeCmd.Dir = mainRepoPath
	if _, err := worktreeCmd.CombinedOutput(); err != nil {
		// If branch already exists, try without -b
		worktreeCmd = exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, fmt.Sprintf("origin/%s", branch))
		worktreeCmd.Dir = mainRepoPath
		if output, retryErr := worktreeCmd.CombinedOutput(); retryErr != nil {
			return nil, fmt.Errorf("failed to create worktree: %s, output: %s", retryErr, output)
		}
	}

	// Apply git config if specified
	if m.gitConfigPath != "" {
		if err := m.applyGitConfig(ctx, worktreePath); err != nil {
			// Non-fatal error
			log.Warn("Failed to apply git config", "error", err)
		}
	}

	// Detect the actual branch name in the worktree
	actualBranch := branch
	branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = worktreePath
	if branchOutput, err := branchCmd.Output(); err == nil {
		detected := strings.TrimSpace(string(branchOutput))
		if detected != "" && detected != "HEAD" {
			actualBranch = detected
		}
	} else {
		log.Warn("Failed to detect actual branch name, falling back to fetch branch", "error", err, "fallback", branch)
	}

	log.Info("Worktree created successfully", "path", worktreePath, "branch", actualBranch)
	return &WorktreeResult{Path: worktreePath, Branch: actualBranch}, nil
}
