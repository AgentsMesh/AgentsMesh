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

func gitCloneWithAuth(ctx context.Context, repoURL, branch, targetDir, authType, credential string) error {
	slog.InfoContext(ctx, "git clone with auth", "auth_type", authType, "branch", branch)
	switch authType {
	case extension.AuthTypeGitHubPAT:
		authedURL, err := injectPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeGitLabPAT:
		authedURL, err := injectGitLabPATIntoURL(repoURL, credential)
		if err != nil {
			return fmt.Errorf("failed to build authenticated URL: %w", err)
		}
		return gitClone(ctx, authedURL, branch, targetDir)

	case extension.AuthTypeSSHKey:
		return gitCloneWithSSHKey(ctx, repoURL, branch, targetDir, credential)

	default:
		return gitClone(ctx, repoURL, branch, targetDir)
	}
}

func injectPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://%s@%s", token, rest), nil
}

// injectGitLabPATIntoURL uses the oauth2 username form GitLab requires.
func injectGitLabPATIntoURL(repoURL, token string) (string, error) {
	if !strings.HasPrefix(repoURL, "https://") {
		return "", fmt.Errorf("PAT auth requires https:// URL, got: %s", repoURL)
	}
	rest := strings.TrimPrefix(repoURL, "https://")
	return fmt.Sprintf("https://oauth2:%s@%s", token, rest), nil
}

func gitCloneWithSSHKey(ctx context.Context, repoURL, branch, targetDir, sshKey string) error {
	isGitSSH := strings.HasPrefix(repoURL, "git@")
	isLocalPath := strings.HasPrefix(repoURL, "/") || strings.HasPrefix(repoURL, ".")
	if !isGitSSH && !isLocalPath {
		return fmt.Errorf("SSH key auth requires git@ URL, got: %s", repoURL)
	}

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

	if err := os.Chmod(tmpKeyFile.Name(), 0600); err != nil {
		return fmt.Errorf("failed to set SSH key permissions: %w", err)
	}

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
		slog.ErrorContext(ctx, "git clone with SSH key failed", "error", err)
		return fmt.Errorf("git clone with SSH key failed: %s: %w", sanitized, err)
	}
	return nil
}

func gitClone(ctx context.Context, url, branch, targetDir string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only https:// URLs are allowed for git clone, got: %s", url)
	}

	if branch != "" {
		if err := validateGitBranch(branch); err != nil {
			return fmt.Errorf("invalid branch: %w", err)
		}
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, "--", url, targetDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		sanitized := sanitizeGitOutput(string(output))
		slog.ErrorContext(ctx, "git clone failed", "branch", branch, "error", err)
		return fmt.Errorf("git clone failed: %s: %w", sanitized, err)
	}
	return nil
}

// sanitizeGitOutput redacts PAT tokens embedded in HTTPS URLs (https://<token>@host).
func sanitizeGitOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "https://"); idx >= 0 {
			if atIdx := strings.Index(line[idx:], "@"); atIdx > 8 {
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
