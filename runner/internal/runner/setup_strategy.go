package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// SetupResult encapsulates the result of a setup strategy execution.
type SetupResult struct {
	WorkingDir string
	BranchName string
	// SandboxRoot overrides the default sandbox root when non-empty.
	// Used by LocalPathStrategy to reuse the source pod's sandbox directory,
	// preventing path template escapes during resume mode.
	SandboxRoot string
}

// SetupStrategy defines the interface for working directory setup strategies.
// Each strategy handles a specific type of sandbox configuration.
// This follows the Strategy Pattern to make setup() extensible without modification (OCP).
type SetupStrategy interface {
	// Name returns the strategy name for logging purposes.
	Name() string

	// CanHandle returns true if this strategy can handle the given configuration.
	CanHandle(cfg *runnerv1.SandboxConfig) bool

	// Setup executes the setup operation.
	// sandboxRoot is the pre-created sandbox root directory.
	// Returns the working directory, branch name, and any error.
	Setup(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (*SetupResult, error)
}

// =============================================================================
// GitWorktreeStrategy - handles repository-based setup with git worktree
// =============================================================================

// GitWorktreeStrategy creates a git worktree for pods with repository configuration.
type GitWorktreeStrategy struct {
	builder *PodBuilder
}

// NewGitWorktreeStrategy creates a new git worktree setup strategy.
func NewGitWorktreeStrategy(b *PodBuilder) *GitWorktreeStrategy {
	return &GitWorktreeStrategy{builder: b}
}

func (s *GitWorktreeStrategy) Name() string {
	return "git_worktree"
}

func (s *GitWorktreeStrategy) CanHandle(cfg *runnerv1.SandboxConfig) bool {
	return cfg != nil && (cfg.HttpCloneUrl != "" || cfg.SshCloneUrl != "")
}

func (s *GitWorktreeStrategy) Setup(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (*SetupResult, error) {
	workingDir, branchName, err := s.builder.setupGitWorktree(ctx, sandboxRoot, cfg)
	if err != nil {
		return nil, err
	}

	// Run preparation script if configured
	if cfg.PreparationScript != "" {
		if err := s.builder.runPreparationScript(ctx, cfg, workingDir, branchName); err != nil {
			return nil, err
		}
	}

	return &SetupResult{WorkingDir: workingDir, BranchName: branchName}, nil
}

// =============================================================================
// LocalPathStrategy - handles resume mode with existing local path
// =============================================================================

// LocalPathStrategy handles pods resuming from an existing local sandbox path.
type LocalPathStrategy struct{}

// NewLocalPathStrategy creates a new local path setup strategy.
func NewLocalPathStrategy() *LocalPathStrategy {
	return &LocalPathStrategy{}
}

func (s *LocalPathStrategy) Name() string {
	return "local_path"
}

func (s *LocalPathStrategy) CanHandle(cfg *runnerv1.SandboxConfig) bool {
	return cfg != nil && cfg.LocalPath != ""
}

func (s *LocalPathStrategy) Setup(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (*SetupResult, error) {
	// Verify local path exists
	if _, err := os.Stat(cfg.LocalPath); os.IsNotExist(err) {
		return nil, &client.PodError{
			Code:    client.ErrCodeWorkDirNotExist,
			Message: fmt.Sprintf("local path does not exist: %s", cfg.LocalPath),
			Details: map[string]string{"path": cfg.LocalPath},
		}
	}

	// For resume mode, the working directory is the workspace inside the existing sandbox
	workingDir := filepath.Join(cfg.LocalPath, "workspace")
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		// Fallback to local path itself if workspace subdirectory doesn't exist
		workingDir = cfg.LocalPath
		logger.Pod().Debug("Workspace subdirectory not found, using local path directly",
			"local_path", cfg.LocalPath)
	}

	return &SetupResult{
		WorkingDir:  workingDir,
		BranchName:  "",
		SandboxRoot: cfg.LocalPath, // Reuse source pod's sandbox to prevent path template escapes
	}, nil
}

// =============================================================================
// EmptySandboxStrategy - handles empty workspace creation (default fallback)
// =============================================================================

// EmptySandboxStrategy creates an empty workspace directory.
// This is the default strategy when no repository or local path is specified.
type EmptySandboxStrategy struct{}

// NewEmptySandboxStrategy creates a new empty sandbox setup strategy.
func NewEmptySandboxStrategy() *EmptySandboxStrategy {
	return &EmptySandboxStrategy{}
}

func (s *EmptySandboxStrategy) Name() string {
	return "empty_sandbox"
}

// CanHandle always returns true as this is the fallback strategy.
func (s *EmptySandboxStrategy) CanHandle(cfg *runnerv1.SandboxConfig) bool {
	return true
}

func (s *EmptySandboxStrategy) Setup(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (*SetupResult, error) {
	workingDir := filepath.Join(sandboxRoot, "workspace")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return nil, &client.PodError{
			Code:    client.ErrCodeSandboxCreate,
			Message: fmt.Sprintf("failed to create temp workspace: %v", err),
		}
	}

	return &SetupResult{WorkingDir: workingDir, BranchName: ""}, nil
}

// =============================================================================
// Default strategies registration
// =============================================================================

// DefaultSetupStrategies returns the default set of setup strategies.
// The order matters: first matching strategy is used.
func DefaultSetupStrategies(b *PodBuilder) []SetupStrategy {
	return []SetupStrategy{
		NewGitWorktreeStrategy(b),
		NewLocalPathStrategy(),
		NewEmptySandboxStrategy(), // Default fallback - must be last
	}
}
