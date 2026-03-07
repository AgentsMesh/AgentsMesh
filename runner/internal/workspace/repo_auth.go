package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/fsutil"
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
		// Bare repository exists, fetch updates using auth URL directly.
		// IMPORTANT: Do NOT persist authURL to remote config — the bare repo is
		// shared across Pods, and any Pod could read the stored token.
		// Instead, pass the auth URL as an explicit fetch argument.
		log.Debug("Repository exists, fetching updates", "path", path)
		authURL := m.prepareAuthURL(repoURL, opts)
		fetchCmd := exec.CommandContext(ctx, "git", "fetch", authURL, "+refs/heads/*:refs/remotes/origin/*")
		fetchCmd.Dir = path
		m.setGitAuthEnv(fetchCmd, opts)
		if output, err := fetchCmd.CombinedOutput(); err != nil {
			// Fetch failed — the bare repo may be corrupted. Remove and re-clone.
			log.Warn("Fetch failed on existing repo, removing corrupted repo and re-cloning",
				"path", path, "error", err, "output", string(output))
			if removeErr := fsutil.RemoveAll(path); removeErr != nil {
				return fmt.Errorf("failed to fetch and failed to remove corrupted repo: fetch error: %s, remove error: %w", output, removeErr)
			}
			return m.cloneBareRepository(ctx, repoURL, path, opts)
		}
		log.Debug("Repository fetched successfully", "path", path)
		return nil
	}

	// Directory may exist but is not a valid bare repo (e.g., previous clone was interrupted).
	// Clean it up before cloning.
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		log.Warn("Directory exists but is not a valid bare repo, removing before clone", "path", path)
		if err := fsutil.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove invalid repo directory: %w", err)
		}
	}

	return m.cloneBareRepository(ctx, repoURL, path, opts)
}

// cloneBareRepository performs a bare clone and configures the repository for worktree usage.
func (m *Manager) cloneBareRepository(ctx context.Context, repoURL, path string, opts *WorktreeOptions) error {
	log := logger.Workspace()

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
		// Clean up any partial clone artifacts to avoid blocking future retries
		if removeErr := fsutil.RemoveAll(path); removeErr != nil {
			log.Warn("Failed to clean up partial clone", "path", path, "error", removeErr)
		}
		return fmt.Errorf("failed to clone: %w, output: %s", err, output)
	}
	log.Debug("Repository cloned successfully", "path", path)

	// After clone, remove token from stored remote URL to prevent cross-pod leakage.
	// The bare repo is shared across pods; only transient operations should use auth.
	cleanURLCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", repoURL)
	cleanURLCmd.Dir = path
	cleanURLCmd.Run() // Best-effort cleanup

	// For bare repos, configure fetch refspec to get all remote branches as origin/*
	// This enables using origin/branch_name references in worktree commands
	configCmd := exec.CommandContext(ctx, "git", "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	configCmd.Dir = path
	configCmd.Run() // Ignore errors

	// Fetch to populate origin/* references using auth URL directly (not stored in config)
	authURL := m.prepareAuthURL(repoURL, opts)
	fetchCmd := exec.CommandContext(ctx, "git", "fetch", authURL, "+refs/heads/*:refs/remotes/origin/*")
	fetchCmd.Dir = path
	m.setGitAuthEnv(fetchCmd, opts)
	fetchCmd.Run() // Ignore errors

	return nil
}

// probeRepositoryAccess tries to find an accessible clone URL using git ls-remote.
// It tests candidate URLs based on available credentials and returns the first one that works.
// This runs with a timeout to avoid blocking if authentication fails.
func (m *Manager) probeRepositoryAccess(ctx context.Context, httpURL, sshURL string, opts *WorktreeOptions) (string, error) {
	log := logger.Workspace()

	type candidate struct {
		url  string
		desc string
	}

	var candidates []candidate

	if opts == nil {
		opts = &WorktreeOptions{}
	}

	// Build candidate list based on available credentials
	if opts.GitToken != "" && httpURL != "" {
		candidates = append(candidates, candidate{url: httpURL, desc: "HTTP+token"})
	}
	if opts.SSHKeyPath != "" && sshURL != "" {
		candidates = append(candidates, candidate{url: sshURL, desc: "SSH+key"})
	}

	// If no credential-matched candidates were found, fall back to trying
	// available URLs with local config (runner_local mode, or mismatched credential/URL)
	if len(candidates) == 0 {
		if sshURL != "" {
			candidates = append(candidates, candidate{url: sshURL, desc: "SSH(local)"})
		}
		if httpURL != "" {
			candidates = append(candidates, candidate{url: httpURL, desc: "HTTP(local)"})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no clone URLs available to probe")
	}

	var errors []string
	for _, c := range candidates {
		log.Debug("Probing repository access", "url", c.url, "method", c.desc)

		probeCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		probeURL := c.url
		// For HTTP URLs with token, embed credentials
		if opts.GitToken != "" && strings.HasPrefix(c.url, "https://") {
			probeURL = m.prepareAuthURL(c.url, opts)
		}

		cmd := exec.CommandContext(probeCtx, "git", "ls-remote", "--exit-code", probeURL, "HEAD")
		m.setProbeEnv(cmd, opts)
		output, err := cmd.CombinedOutput()
		cancel()

		if err == nil {
			log.Info("Repository access probe succeeded", "url", c.url, "method", c.desc)
			return c.url, nil
		}

		// Build detailed error message including both git output and Go error
		outputStr := strings.TrimSpace(string(output))
		errDetail := err.Error()
		if probeCtx.Err() == context.DeadlineExceeded {
			errDetail = "timeout (connection took too long)"
		}
		errMsg := fmt.Sprintf("%s (%s): %s", c.desc, c.url, errDetail)
		if outputStr != "" {
			errMsg = fmt.Sprintf("%s (%s): %s — %s", c.desc, c.url, errDetail, outputStr)
		}
		errors = append(errors, errMsg)
		log.Warn("Repository access probe failed", "url", c.url, "method", c.desc, "error", errDetail, "output", outputStr)
	}

	return "", fmt.Errorf("all repository access methods failed:\n  %s", strings.Join(errors, "\n  "))
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
		return fmt.Errorf("failed to include local config: %w, output: %s", err, output)
	}

	return nil
}
