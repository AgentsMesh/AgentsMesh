package workspace

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/envfilter"
)

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

// setGitAuthEnv sets environment variables for git authentication.
// Always disables interactive prompts to prevent blocking in daemon mode.
func (m *Manager) setGitAuthEnv(cmd *exec.Cmd, opts *WorktreeOptions) {
	// Start with filtered environment (removes sensitive/unnecessary vars)
	env := envfilter.FilterEnv(os.Environ())

	// Always disable interactive prompts — Runner is a daemon and cannot handle interactive input
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_ASKPASS=")

	if opts != nil && opts.SSHKeyPath != "" {
		// Set SSH key path with BatchMode to prevent SSH password prompts.
		// Quote the key path to handle Windows paths with spaces.
		sshCmd := fmt.Sprintf(`ssh -i "%s" -o StrictHostKeyChecking=no -o BatchMode=yes`, opts.SSHKeyPath)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	}

	cmd.Env = env
}

// setProbeEnv sets environment variables for probing repository access.
// Similar to setGitAuthEnv but always forces BatchMode and ConnectTimeout for SSH
// to ensure the probe never blocks.
func (m *Manager) setProbeEnv(cmd *exec.Cmd, opts *WorktreeOptions) {
	env := envfilter.FilterEnv(os.Environ())

	// Disable interactive prompts
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	env = append(env, "GIT_ASKPASS=")

	// Always set SSH command with BatchMode and short timeout for probing
	if opts != nil && opts.SSHKeyPath != "" {
		sshCmd := fmt.Sprintf(`ssh -i "%s" -o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=30`, opts.SSHKeyPath)
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	} else {
		// For runner_local mode, still enforce BatchMode and ConnectTimeout during probe
		env = append(env, "GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=30")
	}

	cmd.Env = env
}

// TempWorkspace creates a temporary workspace directory
func (m *Manager) TempWorkspace(podKey string) string {
	path := filepath.Join(m.root, "temp", podKey)
	os.MkdirAll(path, 0755)
	return path
}

// GetWorkspaceRoot returns the workspace root directory
func (m *Manager) GetWorkspaceRoot() string {
	return m.root
}

// ListWorktrees lists all active worktrees.
// Worktrees are located at sandboxes/{podKey}/worktree
func (m *Manager) ListWorktrees() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sandboxesDir := filepath.Join(m.root, "sandboxes")
	entries, err := os.ReadDir(sandboxesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var worktrees []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		worktreePath := filepath.Join(sandboxesDir, entry.Name(), "worktree")
		// Only include if worktree actually exists
		if _, err := os.Stat(worktreePath); err == nil {
			worktrees = append(worktrees, worktreePath)
		}
	}

	return worktrees, nil
}

// extractRepoName extracts repository name from URL.
// Supports: SCP-style (git@host:user/repo), ssh:// (with optional port), and HTTPS URLs.
func extractRepoName(repoURL string) string {
	// Handle ssh:// URLs: ssh://git@host:port/user/repo.git
	if strings.HasPrefix(repoURL, "ssh://") {
		if u, err := url.Parse(repoURL); err == nil {
			p := strings.TrimPrefix(u.Path, "/")
			p = strings.TrimSuffix(p, ".git")
			if p != "" {
				return strings.ReplaceAll(p, "/", "-")
			}
		}
	}

	// Handle SCP-style SSH URLs: git@github.com:user/repo.git
	if strings.Contains(repoURL, "@") && !strings.Contains(repoURL, "://") {
		if idx := strings.LastIndex(repoURL, ":"); idx != -1 {
			path := repoURL[idx+1:]
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
