package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
)

// setupGitWorktree creates a git worktree for the pod.
func (b *PodBuilder) setupGitWorktree(ctx context.Context, sandboxRoot string, cfg *runnerv1.SandboxConfig) (string, string, error) {
	// Determine repository URL: prefer new fields, fall back to legacy repository_url
	repoURL := cfg.RepositoryUrl
	if cfg.HttpCloneUrl != "" || cfg.SshCloneUrl != "" {
		// New fields are available — use HTTP as primary, probe will select the right one
		if cfg.HttpCloneUrl != "" {
			repoURL = cfg.HttpCloneUrl
		} else {
			repoURL = cfg.SshCloneUrl
		}
	}

	if repoURL == "" {
		return "", "", &client.PodError{
			Code:    client.ErrCodeGitClone,
			Message: "repository_url is required for worktree creation",
		}
	}

	// Use workspace manager if available
	if b.deps.Workspace == nil {
		return "", "", &client.PodError{
			Code:    client.ErrCodeGitWorktree,
			Message: "workspace manager not available for git operations",
		}
	}

	// Report cloning progress
	b.sendProgress("cloning", 30, "Cloning repository...")

	// Build worktree options based on credential type
	opts := []workspace.WorktreeOption{}
	logger.Pod().Debug("Setting up git credentials", "pod_key", b.cmd.PodKey, "credential_type", cfg.CredentialType)

	switch cfg.CredentialType {
	case "runner_local":
		// Use Runner's local git configuration, no credentials needed
		logger.Pod().Debug("Using runner local git config", "pod_key", b.cmd.PodKey)
	case "oauth", "pat":
		// HTTPS + token authentication
		logger.Pod().Debug("Using token authentication", "pod_key", b.cmd.PodKey, "type", cfg.CredentialType)
		if cfg.GitToken != "" {
			opts = append(opts, workspace.WithGitToken(cfg.GitToken))
		}
	case "ssh_key":
		// SSH private key authentication
		if cfg.SshPrivateKey != "" {
			// Write SSH private key to temporary file in sandbox
			keyFile := filepath.Join(sandboxRoot, ".ssh_key")
			if err := os.WriteFile(keyFile, []byte(cfg.SshPrivateKey), 0600); err != nil {
				return "", "", &client.PodError{
					Code:    client.ErrCodeFileCreate,
					Message: fmt.Sprintf("failed to write SSH key: %v", err),
				}
			}
			// On Windows, os.FileMode(0600) is not enforced by the filesystem.
			// SSH clients require strict permissions, so use icacls to remove
			// inherited ACLs and grant read-only access to the current user.
			if runtime.GOOS == "windows" {
				username := os.Getenv("USERNAME")
				if username == "" {
					// Fallback for Windows Service or container environments
					// where USERNAME may not be set.
					if u, err := user.Current(); err == nil {
						username = u.Username
					}
				}
				if username != "" {
					if err := exec.Command("icacls", keyFile, "/inheritance:r",
						"/grant:r", username+":R").Run(); err != nil {
						logger.Pod().Warn("Failed to set SSH key ACL (SSH may reject key if permissions are too open)",
							"error", err, "key_file", keyFile)
					}
				}
			}
			opts = append(opts, workspace.WithSSHKeyPath(keyFile))
			logger.Pod().Debug("SSH key written to sandbox", "pod_key", b.cmd.PodKey, "key_file", keyFile)
		}
	default:
		// Unknown type - fallback to runner_local behavior
		if cfg.CredentialType != "" {
			logger.Pod().Warn("Unknown credential type, using runner local",
				"credential_type", cfg.CredentialType, "pod_key", b.cmd.PodKey)
		}
	}

	// Pass new clone URLs for smart probing
	if cfg.HttpCloneUrl != "" {
		opts = append(opts, workspace.WithHttpCloneURL(cfg.HttpCloneUrl))
	}
	if cfg.SshCloneUrl != "" {
		opts = append(opts, workspace.WithSshCloneURL(cfg.SshCloneUrl))
	}

	// Create git worktree inside sandbox directory: sandboxes/{podKey}/workspace
	workspaceTarget := filepath.Join(sandboxRoot, "workspace")
	result, err := b.deps.Workspace.CreateWorktreeWithOptions(
		ctx,
		repoURL,
		cfg.SourceBranch,
		workspaceTarget,
		opts...,
	)
	if err != nil {
		// Determine error type
		errMsg := err.Error()
		errCode := client.ErrCodeGitWorktree
		if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "Permission denied") {
			errCode = client.ErrCodeGitAuth
		} else if strings.Contains(errMsg, "clone") {
			errCode = client.ErrCodeGitClone
		}
		return "", "", &client.PodError{
			Code:    errCode,
			Message: fmt.Sprintf("failed to create workspace: %v", err),
			Details: map[string]string{
				"repository": cfg.RepositoryUrl,
				"branch":     cfg.SourceBranch,
			},
		}
	}

	// Report progress after successful clone
	b.sendProgress("cloning", 60, "Repository cloned successfully")

	// WorktreeResult.Branch already falls back to the requested branch
	// when detached HEAD is detected, so no additional fallback is needed.
	branchName := result.Branch

	logger.Pod().Info("Git worktree created",
		"pod_key", b.cmd.PodKey,
		"workspace", result.Path,
		"branch", branchName)

	return result.Path, branchName, nil
}
