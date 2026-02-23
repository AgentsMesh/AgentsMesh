package workspace

import (
	"context"
)

// WorkspaceManagerInterface defines the interface for workspace and git worktree management.
// This interface abstracts Manager for testing and decoupling.
type WorkspaceManagerInterface interface {
	// CreateWorktree creates a git worktree for a pod.
	CreateWorktree(ctx context.Context, repoURL, branch, podKey string) (*WorktreeResult, error)

	// CreateWorktreeWithOptions creates a git worktree with custom options.
	CreateWorktreeWithOptions(ctx context.Context, repoURL, branch, worktreePath string, opts ...WorktreeOption) (*WorktreeResult, error)

	// RemoveWorktree removes a git worktree and cleans up associated resources.
	RemoveWorktree(ctx context.Context, worktreePath string) error

	// CleanupOldWorktrees removes stale worktrees that are no longer needed.
	CleanupOldWorktrees(ctx context.Context) error

	// TempWorkspace returns a temporary workspace path for a pod.
	TempWorkspace(podKey string) string

	// GetWorkspaceRoot returns the root directory for workspaces.
	GetWorkspaceRoot() string

	// ListWorktrees returns a list of all worktree paths.
	ListWorktrees() ([]string, error)
}

// Ensure Manager implements WorkspaceManagerInterface.
var _ WorkspaceManagerInterface = (*Manager)(nil)
