package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// gitCommandTimeout is the maximum time allowed for a single git command
const gitCommandTimeout = 5 * time.Second

// GetSandboxStatus returns the sandbox status for a given pod key.
// Sandbox directory structure (created by pod_builder.go):
//
//	{WorkspaceRoot}/
//	├── sandboxes/{podKey}/       # Sandbox = Pod's complete isolated environment
//	│   ├── workspace/            # Workspace directory (with or without git)
//	│   └── .ssh_key              # SSH key (if any)
//	└── repos/{repoName}/         # Bare repository cache (shared)
func (r *Runner) GetSandboxStatus(podKey string) *client.SandboxStatusInfo {
	log := logger.Pod()

	// Sandbox root directory
	sandboxPath := filepath.Join(r.cfg.WorkspaceRoot, "sandboxes", podKey)

	info, err := os.Stat(sandboxPath)
	if os.IsNotExist(err) {
		logger.RunnerTrace().Trace("Sandbox not found", "pod_key", podKey, "path", sandboxPath)
		return &client.SandboxStatusInfo{
			PodKey:    podKey,
			Exists:    false,
			CanResume: false,
		}
	}
	if err != nil {
		log.Error("Failed to stat sandbox directory", "pod_key", podKey, "error", err)
		return &client.SandboxStatusInfo{
			PodKey: podKey,
			Exists: false,
			Error:  err.Error(),
		}
	}

	status := &client.SandboxStatusInfo{
		PodKey:       podKey,
		Exists:       true,
		CanResume:    true, // Sandbox exists, can resume
		SandboxPath:  sandboxPath,
		LastModified: info.ModTime().Unix(),
	}

	// Check for workspace directory inside sandbox and get git info
	workspacePath := filepath.Join(sandboxPath, "workspace")
	if wsInfo, err := os.Stat(workspacePath); err == nil {
		status.LastModified = wsInfo.ModTime().Unix()

		// Get git information if it's a git repository
		gitInfo := r.getGitInfo(workspacePath)
		status.RepositoryURL = gitInfo.RepositoryURL
		status.BranchName = gitInfo.BranchName
		status.CurrentCommit = gitInfo.CurrentCommit
		status.HasUncommittedChanges = gitInfo.HasUncommittedChanges
	}

	// Calculate directory size (optional, can be slow for large directories)
	// Skip for now to avoid blocking the query
	// status.SizeBytes = r.getDirSize(sandboxPath)

	logger.RunnerTrace().Trace("Sandbox status retrieved",
		"pod_key", podKey,
		"exists", status.Exists,
		"sandbox_path", status.SandboxPath,
		"branch", status.BranchName,
		"can_resume", status.CanResume)

	return status
}

// gitInfo holds git repository information
type gitInfo struct {
	RepositoryURL         string
	BranchName            string
	CurrentCommit         string
	HasUncommittedChanges bool
}

// getGitInfo retrieves git information from a workspace directory
func (r *Runner) getGitInfo(workspacePath string) gitInfo {
	info := gitInfo{}

	// Check if it's a git repository
	gitDir := filepath.Join(workspacePath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return info
	}

	// Get remote URL
	if out, err := r.runGitCommand(workspacePath, "remote", "get-url", "origin"); err == nil {
		info.RepositoryURL = strings.TrimSpace(out)
	}

	// Get current branch
	if out, err := r.runGitCommand(workspacePath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.BranchName = strings.TrimSpace(out)
	}

	// Get current commit
	if out, err := r.runGitCommand(workspacePath, "rev-parse", "HEAD"); err == nil {
		info.CurrentCommit = strings.TrimSpace(out)
		// Shorten to 8 characters
		if len(info.CurrentCommit) > 8 {
			info.CurrentCommit = info.CurrentCommit[:8]
		}
	}

	// Check for uncommitted changes
	if out, err := r.runGitCommand(workspacePath, "status", "--porcelain"); err == nil {
		info.HasUncommittedChanges = len(strings.TrimSpace(out)) > 0
	}

	logger.RunnerTrace().Trace("Git info retrieved",
		"path", workspacePath,
		"remote", info.RepositoryURL,
		"branch", info.BranchName,
		"commit", info.CurrentCommit,
		"has_changes", info.HasUncommittedChanges)

	return info
}

// runGitCommand runs a git command in the specified directory with timeout
func (r *Runner) runGitCommand(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// getDirSize calculates the total size of a directory
// This can be slow for large directories, use with caution
func (r *Runner) getDirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
