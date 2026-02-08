package workspace

import (
	"fmt"
	"os"
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

	logger.Runner().Info("Workspace manager created", "root", root)

	return &Manager{
		root:          root,
		gitConfigPath: gitConfigPath,
	}, nil
}
