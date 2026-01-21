package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Manager manages workspace directories and git worktrees
type Manager struct {
	root          string
	gitConfigPath string
	mu            sync.Mutex
}

// WorktreeOptions contains options for creating a worktree
type WorktreeOptions struct {
	GitToken   string // Git token for HTTPS authentication
	SSHKeyPath string // Path to SSH key for SSH authentication
}

// WorktreeOption is a function that modifies WorktreeOptions
type WorktreeOption func(*WorktreeOptions)

// WithGitToken sets the git token for HTTPS authentication
func WithGitToken(token string) WorktreeOption {
	return func(opts *WorktreeOptions) {
		opts.GitToken = token
	}
}

// WithSSHKeyPath sets the SSH key path for SSH authentication
func WithSSHKeyPath(path string) WorktreeOption {
	return func(opts *WorktreeOptions) {
		opts.SSHKeyPath = path
	}
}

// NewManager creates a new workspace manager
func NewManager(root, gitConfigPath string) (*Manager, error) {
	// Ensure root directory exists
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace root: %w", err)
	}

	return &Manager{
		root:          root,
		gitConfigPath: gitConfigPath,
	}, nil
}

// CreateWorktree creates a git worktree for a repository
func (m *Manager) CreateWorktree(ctx context.Context, repoURL, branch, podKey string) (string, error) {
	return m.CreateWorktreeWithOptions(ctx, repoURL, branch, podKey)
}

// CreateWorktreeWithOptions creates a git worktree with additional options
func (m *Manager) CreateWorktreeWithOptions(ctx context.Context, repoURL, branch, podKey string, opts ...WorktreeOption) (string, error) {
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

	// Main repository path
	mainRepoPath := filepath.Join(m.root, "repos", repoName)

	// Clone or fetch the repository with authentication
	if err := m.ensureRepositoryWithAuth(ctx, repoURL, mainRepoPath, options); err != nil {
		return "", fmt.Errorf("failed to ensure repository: %w", err)
	}

	// Worktree path
	worktreePath := filepath.Join(m.root, "worktrees", podKey)

	// Remove existing worktree if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		if err := m.removeWorktreeInternal(ctx, mainRepoPath, worktreePath); err != nil {
			return "", fmt.Errorf("failed to remove existing worktree: %w", err)
		}
	}

	// Create worktree directory
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
	// Use a unique branch name to avoid conflicts
	worktreeBranch := fmt.Sprintf("worktree-%s", podKey)

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
			logger.Workspace().Warn("Failed to apply git config", "error", err)
		}
	}

	return worktreePath, nil
}

// ensureRepository clones or fetches a repository
func (m *Manager) ensureRepository(ctx context.Context, repoURL, path string) error {
	return m.ensureRepositoryWithAuth(ctx, repoURL, path, nil)
}

// ensureRepositoryWithAuth clones or fetches a repository with authentication options
func (m *Manager) ensureRepositoryWithAuth(ctx context.Context, repoURL, path string, opts *WorktreeOptions) error {
	// Check if repository exists (bare repo has HEAD file directly in path, not in .git subdirectory)
	if _, err := os.Stat(filepath.Join(path, "HEAD")); err == nil {
		// Bare repository exists, update remote URL with auth and fetch updates
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
		return nil
	}

	// Clone the repository (bare clone for worktree support)
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

// prepareAuthURL prepares the repository URL with authentication if needed
// For HTTPS URLs with token, embeds token in URL using standard format:
// - Generic: https://x-access-token:TOKEN@host/path (works for GitHub, GitLab, etc.)
func (m *Manager) prepareAuthURL(repoURL string, opts *WorktreeOptions) string {
	if opts == nil || opts.GitToken == "" {
		return repoURL
	}

	// Only modify HTTPS URLs
	if strings.HasPrefix(repoURL, "https://") {
		// Use x-access-token as username - this is a standard format that works with:
		// - GitHub (accepts any username with PAT as password)
		// - GitLab (accepts oauth2 or any username with PAT as password)
		// - Azure DevOps (accepts any username with PAT as password)
		// - Bitbucket (accepts x-token-auth with app password)
		return strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-access-token:%s@", opts.GitToken), 1)
	}

	return repoURL
}

// setGitAuthEnv sets environment variables for git authentication
func (m *Manager) setGitAuthEnv(cmd *exec.Cmd, opts *WorktreeOptions) {
	// Start with current environment
	env := os.Environ()

	if opts == nil {
		cmd.Env = env
		return
	}

	// Set SSH key path if provided
	if opts.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", opts.SSHKeyPath)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	// Disable interactive prompts for HTTPS authentication
	if opts.GitToken != "" {
		env = append(env, "GIT_TERMINAL_PROMPT=0")
	}

	cmd.Env = env
}

// RemoveWorktree removes a worktree
func (m *Manager) RemoveWorktree(ctx context.Context, worktreePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the main repository
	repoPath, err := m.findMainRepo(worktreePath)
	if err != nil {
		// If we can't find the main repo, just remove the directory
		return os.RemoveAll(worktreePath)
	}

	return m.removeWorktreeInternal(ctx, repoPath, worktreePath)
}

// removeWorktreeInternal removes a worktree (internal, no lock)
func (m *Manager) removeWorktreeInternal(ctx context.Context, repoPath, worktreePath string) error {
	// Remove worktree using git
	removeCmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
	removeCmd.Dir = repoPath
	if output, err := removeCmd.CombinedOutput(); err != nil {
		// If git worktree remove fails, try manual removal
		logger.Workspace().Warn("Git worktree remove failed, trying manual removal",
			"error", err, "output", string(output))
		return os.RemoveAll(worktreePath)
	}

	// Prune worktrees
	pruneCmd := exec.CommandContext(ctx, "git", "worktree", "prune")
	pruneCmd.Dir = repoPath
	pruneCmd.Run() // Ignore errors

	return nil
}

// findMainRepo finds the main repository for a worktree
func (m *Manager) findMainRepo(worktreePath string) (string, error) {
	// The .git file in a worktree contains the path to the main repo
	gitPath := filepath.Join(worktreePath, ".git")

	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	// Format: gitdir: /path/to/main/repo/.git/worktrees/name
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "gitdir: ") {
		return "", fmt.Errorf("invalid .git file format")
	}

	gitDir := strings.TrimPrefix(content, "gitdir: ")

	// Navigate up from .git/worktrees/name to .git
	mainGitDir := filepath.Dir(filepath.Dir(gitDir))
	mainRepoDir := filepath.Dir(mainGitDir)

	// For bare repos, the path is different
	if filepath.Base(mainGitDir) == ".git" {
		return mainRepoDir, nil
	}

	// For bare repos, mainGitDir is the repo itself
	return mainGitDir, nil
}

// TempWorkspace creates a temporary workspace directory
func (m *Manager) TempWorkspace(podKey string) string {
	path := filepath.Join(m.root, "temp", podKey)
	os.MkdirAll(path, 0755)
	return path
}

// applyGitConfig applies custom git configuration
func (m *Manager) applyGitConfig(ctx context.Context, repoPath string) error {
	if m.gitConfigPath == "" {
		return nil
	}

	// Read custom config
	data, err := os.ReadFile(m.gitConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read git config: %w", err)
	}

	// Write to local config
	localConfigPath := filepath.Join(repoPath, ".git", "config.local")
	if err := os.WriteFile(localConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local config: %w", err)
	}

	// Include the local config
	cmd := exec.CommandContext(ctx, "git", "config", "--local", "include.path", "config.local")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to include local config: %s, output: %s", err, output)
	}

	return nil
}

// CleanupOldWorktrees removes worktrees older than maxAge
func (m *Manager) CleanupOldWorktrees(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	worktreesDir := filepath.Join(m.root, "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worktreePath := filepath.Join(worktreesDir, entry.Name())

		// Check if worktree is still valid
		if _, err := os.Stat(filepath.Join(worktreePath, ".git")); os.IsNotExist(err) {
			// Invalid worktree, remove it
			os.RemoveAll(worktreePath)
		}
	}

	return nil
}

// extractRepoName extracts repository name from URL
func extractRepoName(repoURL string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.Contains(repoURL, "@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) == 2 {
			path := parts[1]
			path = strings.TrimSuffix(path, ".git")
			return strings.ReplaceAll(path, "/", "-")
		}
	}

	// Handle HTTPS URLs: https://github.com/user/repo.git
	parts := strings.Split(repoURL, "/")
	if len(parts) >= 2 {
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, ".git")
		owner := parts[len(parts)-2]
		return fmt.Sprintf("%s-%s", owner, name)
	}

	return ""
}

// GetWorkspaceRoot returns the workspace root directory
func (m *Manager) GetWorkspaceRoot() string {
	return m.root
}

// ListWorktrees lists all active worktrees
func (m *Manager) ListWorktrees() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	worktreesDir := filepath.Join(m.root, "worktrees")
	entries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var worktrees []string
	for _, entry := range entries {
		if entry.IsDir() {
			worktrees = append(worktrees, filepath.Join(worktreesDir, entry.Name()))
		}
	}

	return worktrees, nil
}
