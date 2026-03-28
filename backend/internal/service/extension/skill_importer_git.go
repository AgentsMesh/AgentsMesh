package extension

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// --- Git helpers ---

// validateGitBranch validates that a branch name contains only safe characters
func validateGitBranch(branch string) error {
	for _, c := range branch {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/' {
			continue
		}
		return fmt.Errorf("invalid branch name character: %c", c)
	}
	return nil
}

// gitCloneWithAuth clones a repo using the specified auth method.
// For PAT-based auth, the token is injected into the URL so git receives it via HTTPS.
// For SSH key auth, a temporary identity file is created and passed via GIT_SSH_COMMAND.
func gitCloneWithAuth(ctx context.Context, repoURL, branch, targetDir, authType, credential string) error {
	slog.Info("git clone with auth", "auth_type", authType, "branch", branch)
	switch authType {
	case extension.AuthTypeGitHubPAT:
		// GitHub PAT: inject as https://<token>@github.com/owner/repo.git
		authedURL, err := injectPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeGitLabPAT:
		// GitLab PAT: inject as https://oauth2:<token>@gitlab.com/owner/repo.git
		authedURL, err := injectGitLabPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeSSHKey:
		return gitCloneWithSSHKey(ctx, repoURL, branch, targetDir, credential)

	default:
		// Fall back to unauthenticated clone
		return gitClone(ctx, repoURL, branch, targetDir)
	}
}

// injectPATIntoURL inserts a GitHub PAT into an HTTPS URL.
// https://github.com/owner/repo -> https://<token>@github.com/owner/repo
func injectPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	// Strip "https://" and prepend token
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://%s@%s", token, rest), nil
}

// injectGitLabPATIntoURL inserts a GitLab PAT using oauth2 username.
// https://gitlab.com/owner/repo -> https://oauth2:<token>@gitlab.com/owner/repo
func injectGitLabPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://oauth2:%s@%s", token, rest), nil
}

// gitCloneWithSSHKey clones using a temporary SSH identity file.
// Only git@ SSH URLs and local paths are allowed to prevent SSRF via arbitrary protocols.
func gitCloneWithSSHKey(ctx context.Context, repoURL, branch, targetDir, sshKey string) error {
	// Validate URL format -- allow git@ SSH URLs and local filesystem paths,
	// but reject http/https/ftp protocols to prevent SSRF.
	isGitSSH := strings.HasPrefix(repoURL, "git@")
	isLocalPath := strings.HasPrefix(repoURL, "/") || strings.HasPrefix(repoURL, ".")
	if !isGitSSH && !isLocalPath {
		return fmt.Errorf("SSH key auth requires git@ URL, got: %s", repoURL)
	}

	// Write SSH key to temporary file
	tmpKeyFile, err := os.CreateTemp("", "skill-ssh-key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp SSH key file: %w", err)
	}
	defer os.Remove(tmpKeyFile.Name())

	if _, err := tmpKeyFile.WriteString(sshKey); err != nil {
		tmpKeyFile.Close()
		return fmt.Errorf("failed to write SSH key: %w", err)
	}
	tmpKeyFile.Close()

	// Set proper permissions (600)
	if err := os.Chmod(tmpKeyFile.Name(), 0600); err != nil {
		return fmt.Errorf("failed to set SSH key permissions: %w", err)
	}

	// Validate branch name if provided
	if branch != "" {
		if err := validateGitBranch(branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", repoURL, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	sshCommand := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", tmpKeyFile.Name())
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND="+sshCommand,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		sanitized := sanitizeGitOutput(string(output))
		slog.Error("git clone with SSH key failed", "error", err)
		return fmt.Errorf("git clone with SSH key failed: %s: %w", sanitized, err)
	}
	return nil
}

func gitClone(ctx context.Context, url, branch, targetDir string) error {
	// Validate URL scheme - only allow https://
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only https:// URLs are allowed for git clone, got: %s", url)
	}

	// Validate branch name if provided
	if branch != "" {
		if err := validateGitBranch(branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	// Use -- separator to prevent argument injection
	args = append(args, "--", url, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Sanitize output to prevent PAT tokens from leaking into logs/errors
		sanitized := sanitizeGitOutput(string(output))
		slog.Error("git clone failed", "branch", branch, "error", err)
		return fmt.Errorf("git clone failed: %s: %w", sanitized, err)
	}
	return nil
}

// sanitizeGitOutput removes potential credentials from git command output.
// PAT tokens embedded in HTTPS URLs (e.g. https://<token>@github.com) are redacted.
func sanitizeGitOutput(output string) string {
	// Redact HTTPS URLs that contain embedded credentials
	// Pattern: https://<anything>@<host>
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "https://"); idx >= 0 {
			if atIdx := strings.Index(line[idx:], "@"); atIdx > 8 {
				// Redact the credential portion
				lines[i] = line[:idx] + "https://[REDACTED]" + line[idx+atIdx:]
			}
		}
	}
	return strings.Join(lines, "\n")
}

func gitHead(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
