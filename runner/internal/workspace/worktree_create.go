package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// CreateWorktree creates a git worktree for a repository.
// The worktree is created inside the sandbox directory: sandboxes/{podKey}/workspace
func (m *Manager) CreateWorktree(ctx context.Context, repoURL, branch, podKey string) (string, error) {
	workspacePath := filepath.Join(m.root, "sandboxes", podKey, "workspace")
	return m.CreateWorktreeWithOptions(ctx, repoURL, branch, workspacePath)
}

// CreateWorktreeWithOptions creates a git worktree with additional options.
// worktreePath is the full path where the worktree should be created.
func (m *Manager) CreateWorktreeWithOptions(ctx context.Context, repoURL, branch, worktreePath string, opts ...WorktreeOption) (string, error) {
	log := logger.Workspace()
	log.Info("Creating worktree", "repo", repoURL, "branch", branch, "path", worktreePath)

	// Apply options
	options := &WorktreeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse repo name from URL
	repoName := extractRepoName(repoURL)
	if repoName == "" {
		return "", fmt.Errorf("invalid repository URL: %s", repoURL)
	}
	log.Debug("Parsed repo name", "name", repoName)

	// Main repository path (bare repo cache, shared across pods)
	mainRepoPath := filepath.Join(m.root, "repos", repoName)

	// Clone or fetch the repository with authentication
	if err := m.ensureRepositoryWithAuth(ctx, repoURL, mainRepoPath, options); err != nil {
		return "", fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Remove existing worktree if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		if err := m.removeWorktreeInternal(ctx, mainRepoPath, worktreePath); err != nil {
			return "", fmt.Errorf("failed to remove existing worktree: %w", err)
		}
	}

	// Create worktree parent directory
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree parent dir: %w", err)
	}

	// Fetch the branch
	if branch == "" {
		branch = "main"
	}

	// Fetch from remote
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin", branch)
	fetchCmd.Dir = mainRepoPath
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		// Try 'master' if 'main' fails
		if branch == "main" {
			branch = "master"
			fetchCmd = exec.CommandContext(ctx, "git", "fetch", "origin", branch)
			fetchCmd.Dir = mainRepoPath
			if output, err = fetchCmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("failed to fetch branch: %s, output: %s", err, output)
			}
		} else {
			return "", fmt.Errorf("failed to fetch branch: %s, output: %s", err, output)
		}
	}

	// Create worktree
	// Use a unique branch name based on parent directory name (sandbox podKey)
	// e.g., /path/sandboxes/pod-123/worktree -> worktree-pod-123
	parentDir := filepath.Base(filepath.Dir(worktreePath))
	worktreeBranch := fmt.Sprintf("worktree-%s", parentDir)

	worktreeCmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", worktreeBranch, worktreePath, fmt.Sprintf("origin/%s", branch))
	worktreeCmd.Dir = mainRepoPath
	if output, err := worktreeCmd.CombinedOutput(); err != nil {
		// If branch already exists, try without -b
		worktreeCmd = exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, fmt.Sprintf("origin/%s", branch))
		worktreeCmd.Dir = mainRepoPath
		if output, err = worktreeCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to create worktree: %s, output: %s", err, output)
		}
	}

	// Apply git config if specified
	if m.gitConfigPath != "" {
		if err := m.applyGitConfig(ctx, worktreePath); err != nil {
			// Non-fatal error
			log.Warn("Failed to apply git config", "error", err)
		}
	}

	log.Info("Worktree created successfully", "path", worktreePath, "branch", branch)
	return worktreePath, nil
}
