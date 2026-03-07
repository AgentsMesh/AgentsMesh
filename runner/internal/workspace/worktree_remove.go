package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/fsutil"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// RemoveWorktree removes a worktree
func (m *Manager) RemoveWorktree(ctx context.Context, worktreePath string) error {
	log := logger.Workspace()
	log.Info("Removing worktree", "path", worktreePath)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the main repository
	repoPath, err := m.findMainRepo(worktreePath)
	if err != nil {
		// If we can't find the main repo, just remove the directory
		log.Debug("Main repo not found, removing directory directly", "path", worktreePath)
		return fsutil.RemoveAll(worktreePath)
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
		return fsutil.RemoveAll(worktreePath)
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

// CleanupOldWorktrees removes invalid worktrees from sandboxes.
// Worktrees are located at sandboxes/{podKey}/worktree
func (m *Manager) CleanupOldWorktrees(ctx context.Context) error {
	log := logger.Workspace()
	log.Info("Starting worktree cleanup")

	m.mu.Lock()
	defer m.mu.Unlock()

	sandboxesDir := filepath.Join(m.root, "sandboxes")
	entries, err := os.ReadDir(sandboxesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cleanedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worktreePath := filepath.Join(sandboxesDir, entry.Name(), "worktree")

		// Check if worktree exists and is valid
		if _, err := os.Stat(worktreePath); err == nil {
			// Worktree exists, check if it's still valid
			if _, err := os.Stat(filepath.Join(worktreePath, ".git")); os.IsNotExist(err) {
				// Invalid worktree (no .git), remove it
				if err := fsutil.RemoveAll(worktreePath); err != nil {
					log.Warn("Failed to remove invalid worktree", "path", worktreePath, "error", err)
				} else {
					cleanedCount++
				}
			}
		}
	}

	log.Info("Worktree cleanup completed", "cleaned_count", cleanedCount)
	return nil
}
