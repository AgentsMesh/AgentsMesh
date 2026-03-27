package extension

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// gitCloneWithAuth
// =============================================================================

func TestGitCloneWithAuth_GitHubPAT(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "https://github.com/owner/repo.git", "", targetDir, extension.AuthTypeGitHubPAT, "ghp_test123")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestGitCloneWithAuth_GitLabPAT(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "https://gitlab.com/owner/repo.git", "", targetDir, extension.AuthTypeGitLabPAT, "glpat-test456")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestGitCloneWithAuth_SSHKey(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "git@github.com:owner/repo.git", "", targetDir, extension.AuthTypeSSHKey, "fake-ssh-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithAuth_UnknownType(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "https://github.com/owner/repo.git", "", targetDir, "unknown_type", "some_cred")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
}

func TestGitCloneWithAuth_PATInjectError(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "http://github.com/owner/repo.git", "", targetDir, extension.AuthTypeGitHubPAT, "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build authenticated URL")
}

func TestGitCloneWithAuth_GitLabPATInjectError(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "http://gitlab.com/owner/repo.git", "", targetDir, extension.AuthTypeGitLabPAT, "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build authenticated URL")
}

// =============================================================================
// validateGitBranch — additional edge cases
// =============================================================================

func TestValidateGitBranch_ValidChars(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{"lowercase letters", "abcdefghijklmnopqrstuvwxyz"},
		{"uppercase letters", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"digits", "0123456789"},
		{"hyphens", "my-branch"},
		{"underscores", "my_branch"},
		{"dots", "release.1.0"},
		{"slashes", "feature/my-branch/sub"},
		{"mixed valid chars", "Release-1.0_beta/v2.3"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitBranch(tt.branch)
			assert.NoError(t, err)
		})
	}
}

func TestValidateGitBranch_InvalidChars(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{"space", "branch name"},
		{"at sign", "branch@name"},
		{"hash", "branch#name"},
		{"exclamation", "branch!name"},
		{"question mark", "branch?name"},
		{"tilde", "branch~name"},
		{"caret", "branch^name"},
		{"colon", "branch:name"},
		{"backslash", "branch\\name"},
		{"curly brace", "branch{name"},
		{"square bracket", "branch[name"},
		{"star", "branch*name"},
		{"newline", "branch\nname"},
		{"tab", "branch\tname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitBranch(tt.branch)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid branch name character")
		})
	}
}

// =============================================================================
// gitCloneWithSSHKey — additional coverage
// =============================================================================

func TestGitCloneWithSSHKey_SSHKeyFileContents(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	sshKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key-content\n-----END OPENSSH PRIVATE KEY-----"

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "", targetDir, sshKey)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone with SSH key failed",
		"error should come from git clone, not from SSH key file handling")
}

func TestGitCloneWithSSHKey_EmptyBranch_NoBranchArg(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/nonexistent.git", "", targetDir, "fake-ssh-key")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "invalid branch")
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithSSHKey_NonEmptyBranch_PassesBranchArg(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/nonexistent.git", "main", targetDir, "fake-ssh-key")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "invalid branch")
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithSSHKey_InvalidBranch_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "branch name with spaces", targetDir, "fake-ssh-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch")
}

func TestGitCloneWithSSHKey_VerifiesKeyFilePermissions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	targetDir := t.TempDir()
	sshKey := "test-ssh-key-content"

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "", targetDir, sshKey)
	require.Error(t, err)
}

func TestGitCloneWithSSHKey_SuccessfulClone_LocalRepo(t *testing.T) {
	sourceDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = sourceDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git init/config failed: %s", string(out))
	}

	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("hello"), 0644))
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourceDir
	require.NoError(t, addCmd.Run())
	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = sourceDir
	require.NoError(t, commitCmd.Run())

	targetDir := filepath.Join(t.TempDir(), "cloned")
	err := gitCloneWithSSHKey(context.Background(), sourceDir, "", targetDir, "fake-ssh-key")
	require.NoError(t, err, "gitCloneWithSSHKey should succeed for local repo")

	assert.True(t, fileExists(filepath.Join(targetDir, "README.md")))
}

func TestGitCloneWithSSHKey_SuccessfulClone_WithBranch(t *testing.T) {
	sourceDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = sourceDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git setup failed: %s", string(out))
	}

	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0644))
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourceDir
	require.NoError(t, addCmd.Run())
	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = sourceDir
	require.NoError(t, commitCmd.Run())

	branchCmd := exec.Command("git", "checkout", "-b", "feature/test")
	branchCmd.Dir = sourceDir
	out, err := branchCmd.CombinedOutput()
	require.NoError(t, err, "branch create failed: %s", string(out))

	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "branch-file.txt"), []byte("branch content"), 0644))
	addCmd2 := exec.Command("git", "add", ".")
	addCmd2.Dir = sourceDir
	require.NoError(t, addCmd2.Run())
	commitCmd2 := exec.Command("git", "commit", "-m", "branch commit")
	commitCmd2.Dir = sourceDir
	require.NoError(t, commitCmd2.Run())

	targetDir := filepath.Join(t.TempDir(), "cloned")
	err = gitCloneWithSSHKey(context.Background(), sourceDir, "feature/test", targetDir, "fake-ssh-key")
	require.NoError(t, err, "gitCloneWithSSHKey should succeed with branch for local repo")

	assert.True(t, fileExists(filepath.Join(targetDir, "branch-file.txt")))
}
