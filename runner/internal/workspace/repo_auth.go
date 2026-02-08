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

// ensureRepository clones or fetches a repository
func (m *Manager) ensureRepository(ctx context.Context, repoURL, path string) error {
	return m.ensureRepositoryWithAuth(ctx, repoURL, path, nil)
}

// ensureRepositoryWithAuth clones or fetches a repository with authentication options
func (m *Manager) ensureRepositoryWithAuth(ctx context.Context, repoURL, path string, opts *WorktreeOptions) error {
	log := logger.Workspace()

	// Check if repository exists (bare repo has HEAD file directly in path, not in .git subdirectory)
	if _, err := os.Stat(filepath.Join(path, "HEAD")); err == nil {
		// Bare repository exists, update remote URL with auth and fetch updates
		log.Debug("Repository exists, fetching updates", "path", path)
		// Update remote URL with authentication (for fetch operations)
		authURL := m.prepareAuthURL(repoURL, opts)
		setURLCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", authURL)
		setURLCmd.Dir = path
		setURLCmd.Run() // Ignore errors, URL might already be set

		fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all")
		fetchCmd.Dir = path
		m.setGitAuthEnv(fetchCmd, opts)
		if output, err := fetchCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fetch: %s, output: %s", err, output)
		}
		log.Debug("Repository fetched successfully", "path", path)
		return nil
	}

	// Clone the repository (bare clone for worktree support)
	log.Debug("Cloning repository", "url", repoURL, "path", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create repo parent dir: %w", err)
	}

	// Prepare clone URL with token if provided
	cloneURL := m.prepareAuthURL(repoURL, opts)

	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--bare", cloneURL, path)
	m.setGitAuthEnv(cloneCmd, opts)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone: %s, output: %s", err, output)
	}
	log.Debug("Repository cloned successfully", "path", path)

	// For bare repos, configure fetch refspec to get all remote branches as origin/*
	// This enables using origin/branch_name references in worktree commands
	configCmd := exec.CommandContext(ctx, "git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	configCmd.Dir = path
	configCmd.Run() // Ignore errors

	// Fetch to populate origin/* references
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", "--all")
	fetchCmd.Dir = path
	m.setGitAuthEnv(fetchCmd, opts)
	fetchCmd.Run() // Ignore errors

	return nil
}

// applyGitConfig applies custom git configuration to a worktree.
// In a worktree, .git is a file pointing to the main repo, so we use
// git config --local which handles this correctly.
func (m *Manager) applyGitConfig(ctx context.Context, worktreePath string) error {
	if m.gitConfigPath == "" {
		return nil
	}

	// Read custom config
	data, err := os.ReadFile(m.gitConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read git config: %w", err)
	}

	// Get the actual git directory for this worktree
	// In a worktree, `git rev-parse --git-dir` returns the correct .git directory
	gitDirCmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	gitDirCmd.Dir = worktreePath
	gitDirOutput, err := gitDirCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git directory: %w", err)
	}
	gitDir := strings.TrimSpace(string(gitDirOutput))

	// Make path absolute if relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(worktreePath, gitDir)
	}

	// Write to local config in the actual git directory
	localConfigPath := filepath.Join(gitDir, "config.local")
	if err := os.WriteFile(localConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local config: %w", err)
	}

	// Include the local config
	cmd := exec.CommandContext(ctx, "git", "config", "--local", "include.path", "config.local")
	cmd.Dir = worktreePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to include local config: %s, output: %s", err, output)
	}

	return nil
}
