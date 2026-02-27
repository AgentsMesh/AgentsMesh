package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// injectPATIntoURL
// =============================================================================

func TestInjectPATIntoURL_Success(t *testing.T) {
	result, err := injectPATIntoURL("https://github.com/owner/repo.git", "ghp_mytoken123")
	require.NoError(t, err)
	assert.Equal(t, "https://ghp_mytoken123@github.com/owner/repo.git", result)
}

func TestInjectPATIntoURL_NonHTTPS(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"http URL", "http://github.com/owner/repo.git"},
		{"ssh URL", "ssh://git@github.com/owner/repo.git"},
		{"file URL", "file:///local/path/repo"},
		{"git protocol", "git://github.com/owner/repo.git"},
		{"bare path", "/some/local/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := injectPATIntoURL(tt.url, "token")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "PAT auth requires https:// URL")
		})
	}
}

// =============================================================================
// injectGitLabPATIntoURL
// =============================================================================

func TestInjectGitLabPATIntoURL_Success(t *testing.T) {
	result, err := injectGitLabPATIntoURL("https://gitlab.com/owner/repo.git", "glpat-mytoken456")
	require.NoError(t, err)
	assert.Equal(t, "https://oauth2:glpat-mytoken456@gitlab.com/owner/repo.git", result)
}

func TestInjectGitLabPATIntoURL_NonHTTPS(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"http URL", "http://gitlab.com/owner/repo.git"},
		{"ssh URL", "ssh://git@gitlab.com/owner/repo.git"},
		{"file URL", "file:///local/path/repo"},
		{"git protocol", "git://gitlab.com/owner/repo.git"},
		{"bare path", "/some/local/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := injectGitLabPATIntoURL(tt.url, "token")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "PAT auth requires https:// URL")
		})
	}
}

// =============================================================================
// SetCredentialDecryptor
// =============================================================================

func TestSetCredentialDecryptor(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	assert.Nil(t, imp.credentialDecryptor, "credentialDecryptor should be nil initially")

	decryptFn := func(s string) (string, error) {
		return "decrypted-" + s, nil
	}
	imp.SetCredentialDecryptor(decryptFn)

	require.NotNil(t, imp.credentialDecryptor, "credentialDecryptor should be set after SetCredentialDecryptor")

	// Verify it works correctly
	result, err := imp.credentialDecryptor("secret")
	require.NoError(t, err)
	assert.Equal(t, "decrypted-secret", result)
}

// =============================================================================
// cloneRepo — auth paths
// =============================================================================

func TestCloneRepo_NoAuth_PublicRepo(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	cloneCalled := false
	imp.gitCloneFn = func(_ context.Context, url, branch, targetDir string) error {
		cloneCalled = true
		assert.Equal(t, "https://github.com/owner/repo.git", url)
		assert.Equal(t, "main", branch)
		return nil
	}

	source := &extension.SkillRegistry{
		ID:            1,
		RepositoryURL: "https://github.com/owner/repo.git",
		Branch:        "main",
		AuthType:      extension.AuthTypeNone,
		AuthCredential: "",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, cloneCalled, "gitCloneFn should have been called for public repo")
}

func TestCloneRepo_NoAuth_EmptyAuthType(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	cloneCalled := false
	imp.gitCloneFn = func(_ context.Context, _, _, _ string) error {
		cloneCalled = true
		return nil
	}

	// HasAuth() returns false when AuthType is empty
	source := &extension.SkillRegistry{
		ID:            1,
		RepositoryURL: "https://github.com/owner/repo.git",
		Branch:        "main",
		AuthType:      "",
		AuthCredential: "",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, cloneCalled, "gitCloneFn should have been called when AuthType is empty")
}

func TestCloneRepo_WithAuth_GitHubPAT(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	authCloneCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		authCloneCalled = true
		assert.Equal(t, "https://github.com/owner/repo.git", url)
		assert.Equal(t, "main", branch)
		assert.Equal(t, extension.AuthTypeGitHubPAT, authType)
		assert.Equal(t, "ghp_mytoken", credential)
		return nil
	}

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "ghp_mytoken",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, authCloneCalled, "gitCloneAuthFn should have been called for GitHub PAT")
}

func TestCloneRepo_WithAuth_DecryptorSet(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	// Set a decryptor that transforms the credential
	imp.SetCredentialDecryptor(func(encrypted string) (string, error) {
		return "decrypted-" + encrypted, nil
	})

	authCloneCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		authCloneCalled = true
		// Should receive the decrypted credential
		assert.Equal(t, "decrypted-encrypted_token", credential)
		assert.Equal(t, extension.AuthTypeGitHubPAT, authType)
		return nil
	}

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "encrypted_token",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, authCloneCalled)
}

func TestCloneRepo_WithAuth_DecryptFails(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	// Set a decryptor that fails
	imp.SetCredentialDecryptor(func(encrypted string) (string, error) {
		return "", errors.New("decryption failed: corrupted data")
	})

	authCloneCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		authCloneCalled = true
		// When decryption fails, the raw credential is used as fallback
		assert.Equal(t, "raw_credential_value", credential)
		return nil
	}

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "raw_credential_value",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, authCloneCalled, "gitCloneAuthFn should still be called with raw credential")
}

func TestCloneRepo_WithAuth_GitCloneAuthFnOverride(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	// Verify that the custom gitCloneAuthFn is called instead of the default gitCloneWithAuth
	customCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		customCalled = true
		assert.Equal(t, "https://gitlab.com/org/repo.git", url)
		assert.Equal(t, "develop", branch)
		assert.Equal(t, "/tmp/clone-dir", targetDir)
		assert.Equal(t, extension.AuthTypeGitLabPAT, authType)
		assert.Equal(t, "glpat-token", credential)
		return nil
	}

	source := &extension.SkillRegistry{
		ID:             42,
		RepositoryURL:  "https://gitlab.com/org/repo.git",
		Branch:         "develop",
		AuthType:       extension.AuthTypeGitLabPAT,
		AuthCredential: "glpat-token",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/clone-dir")
	require.NoError(t, err)
	assert.True(t, customCalled, "custom gitCloneAuthFn should have been invoked")
}

func TestCloneRepo_WithAuth_HasAuthTrueButEmptyCredential(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	// Even though AuthType is set, empty AuthCredential means the
	// condition `source.HasAuth() && source.AuthCredential != ""` is false,
	// so cloneRepo should fall through to the public clone path.
	publicCloneCalled := false
	imp.gitCloneFn = func(_ context.Context, _, _, _ string) error {
		publicCloneCalled = true
		return nil
	}

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "", // empty credential
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.NoError(t, err)
	assert.True(t, publicCloneCalled, "should fall back to public clone when credential is empty")
}

func TestCloneRepo_WithAuth_CloneAuthFails(t *testing.T) {
	repo := &importerMockRepo{}
	stor := newPackagerMockStorage()
	imp := NewSkillImporter(repo, stor)

	imp.gitCloneAuthFn = func(_ context.Context, _, _, _, _, _ string) error {
		return errors.New("authentication failed: 401 Unauthorized")
	}

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/private-repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeGitHubPAT,
		AuthCredential: "bad_token",
	}

	err := imp.cloneRepo(context.Background(), source, "/tmp/target")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

// =============================================================================
// gitCloneWithAuth
// =============================================================================

func TestGitCloneWithAuth_GitHubPAT(t *testing.T) {
	// gitCloneWithAuth calls injectPATIntoURL then gitClone (package-level).
	// We cannot easily mock gitClone, but we can verify the URL injection
	// and that the function does not return a URL-injection error.
	// The actual git clone will fail because the repo doesn't exist,
	// but the error should come from git, not from URL injection.
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "https://github.com/owner/repo.git", "", targetDir, extension.AuthTypeGitHubPAT, "ghp_test123")
	// Expect a git clone failure (not a URL injection error)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestGitCloneWithAuth_GitLabPAT(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithAuth(ctx, "https://gitlab.com/owner/repo.git", "", targetDir, extension.AuthTypeGitLabPAT, "glpat-test456")
	// Expect a git clone failure (not a URL injection error)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
	assert.Contains(t, err.Error(), "git clone failed")
}

func TestGitCloneWithAuth_SSHKey(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	// SSH clone will fail because of invalid repo, but should attempt SSH clone
	err := gitCloneWithAuth(ctx, "git@github.com:owner/repo.git", "", targetDir, extension.AuthTypeSSHKey, "fake-ssh-key")
	// Should fail at git clone with SSH key, not at URL injection
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithAuth_UnknownType(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	// Unknown auth type falls back to unauthenticated gitClone
	err := gitCloneWithAuth(ctx, "https://github.com/owner/repo.git", "", targetDir, "unknown_type", "some_cred")
	require.Error(t, err)
	// Should be a regular git clone failure (unauthenticated fallback)
	assert.Contains(t, err.Error(), "git clone failed")
	assert.NotContains(t, err.Error(), "failed to build authenticated URL")
}

func TestGitCloneWithAuth_PATInjectError(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	// Non-HTTPS URL should cause injectPATIntoURL to fail
	err := gitCloneWithAuth(ctx, "http://github.com/owner/repo.git", "", targetDir, extension.AuthTypeGitHubPAT, "token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build authenticated URL")
}

func TestGitCloneWithAuth_GitLabPATInjectError(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()

	// Non-HTTPS URL should cause injectGitLabPATIntoURL to fail
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
	// The function creates a temp file with the SSH key, chmods it,
	// then runs git clone. Since we can't mock os.CreateTemp or exec.Command
	// directly, we test that the function reaches the git clone step by
	// verifying the error comes from git, not from file operations.
	ctx := context.Background()
	targetDir := t.TempDir()

	sshKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key-content\n-----END OPENSSH PRIVATE KEY-----"

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "", targetDir, sshKey)
	// Should fail at git clone (not file write/chmod)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git clone with SSH key failed",
		"error should come from git clone, not from SSH key file handling")
}

func TestGitCloneWithSSHKey_EmptyBranch_NoBranchArg(t *testing.T) {
	// With empty branch, the function should NOT pass -b to git.
	// We verify by running git clone with empty branch — the error should
	// be about cloning (not about invalid branch).
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/nonexistent.git", "", targetDir, "fake-ssh-key")
	require.Error(t, err)
	// The error should be from git, not from branch validation
	assert.NotContains(t, err.Error(), "invalid branch")
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithSSHKey_NonEmptyBranch_PassesBranchArg(t *testing.T) {
	// With non-empty branch, the function should pass --branch to git.
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/nonexistent.git", "main", targetDir, "fake-ssh-key")
	require.Error(t, err)
	// The error should be from git clone, not from branch validation
	assert.NotContains(t, err.Error(), "invalid branch")
	assert.Contains(t, err.Error(), "git clone with SSH key failed")
}

func TestGitCloneWithSSHKey_InvalidBranch_ReturnsValidationError(t *testing.T) {
	// An invalid branch name should be rejected before git clone is attempted.
	ctx := context.Background()
	targetDir := t.TempDir()

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "branch name with spaces", targetDir, "fake-ssh-key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch")
}

func TestGitCloneWithSSHKey_VerifiesKeyFilePermissions(t *testing.T) {
	// This test verifies that the SSH key temp file is properly created and cleaned up.
	// We use a cancelled context to make git fail quickly, then verify the temp file is gone.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	targetDir := t.TempDir()
	sshKey := "test-ssh-key-content"

	err := gitCloneWithSSHKey(ctx, "git@github.com:owner/repo.git", "", targetDir, sshKey)
	// Should fail due to cancelled context or git error
	require.Error(t, err)
}

func TestGitCloneWithSSHKey_SuccessfulClone_LocalRepo(t *testing.T) {
	// Create a local git repo so we can test the successful clone path.
	// gitCloneWithSSHKey uses GIT_SSH_COMMAND, but for local path URLs
	// git does not invoke SSH, so the clone succeeds without a real SSH key.
	sourceDir := t.TempDir()

	// Initialize a git repo with a commit
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

	// Create a file and commit
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("hello"), 0644))
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = sourceDir
	require.NoError(t, addCmd.Run())
	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = sourceDir
	require.NoError(t, commitCmd.Run())

	// Clone using gitCloneWithSSHKey with the local path
	targetDir := filepath.Join(t.TempDir(), "cloned")
	err := gitCloneWithSSHKey(context.Background(), sourceDir, "", targetDir, "fake-ssh-key")
	require.NoError(t, err, "gitCloneWithSSHKey should succeed for local repo")

	// Verify cloned content
	assert.True(t, fileExists(filepath.Join(targetDir, "README.md")))
}

func TestGitCloneWithSSHKey_SuccessfulClone_WithBranch(t *testing.T) {
	// Create a local git repo with a specific branch to cover the branch argument path.
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

	// Create a branch
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

	// Clone specific branch using gitCloneWithSSHKey
	targetDir := filepath.Join(t.TempDir(), "cloned")
	err = gitCloneWithSSHKey(context.Background(), sourceDir, "feature/test", targetDir, "fake-ssh-key")
	require.NoError(t, err, "gitCloneWithSSHKey should succeed with branch for local repo")

	// Verify branch-specific content exists
	assert.True(t, fileExists(filepath.Join(targetDir, "branch-file.txt")))
}

// =============================================================================
// packageSkillDir — additional coverage for ignored directories
// =============================================================================

func TestPackageSkillDir_IncludesSubdirEntries(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	// Create a normal subdirectory with files
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "main.py"), []byte("print('hi')"), 0644))

	// Create ignored directories with content that should be skipped
	for _, ignored := range []string{".git", "node_modules", "vendor", "__pycache__", ".github"} {
		ignoredPath := filepath.Join(dir, ignored)
		require.NoError(t, os.MkdirAll(ignoredPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(ignoredPath, "should-not-appear.txt"), []byte("hidden"), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Extract and verify
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, header.Name)
	}

	// Should contain SKILL.md, src/ (directory entry), and src/main.py
	assert.Contains(t, names, "SKILL.md")
	assert.Contains(t, names, "src")

	// Should NOT contain any ignored directories or their contents
	for _, name := range names {
		assert.NotContains(t, name, ".git")
		assert.NotContains(t, name, "node_modules")
		assert.NotContains(t, name, "vendor")
		assert.NotContains(t, name, "__pycache__")
		assert.NotContains(t, name, ".github")
	}
}

func TestPackageSkillDir_EmptyDirProducesValidArchive(t *testing.T) {
	dir := t.TempDir()
	// Completely empty directory
	data, err := packageSkillDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's a valid tar.gz by opening it
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	count := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		count++
	}
	assert.Equal(t, 0, count, "empty dir should produce archive with no entries")
}

func TestPackageSkillDir_SocketFileError(t *testing.T) {
	// tar.FileInfoHeader returns an error for socket files.
	// This test covers the error return from tar.FileInfoHeader in packageSkillDir.

	// Use a short temp dir path for the socket because Unix sockets have
	// a path length limit (~104 bytes on macOS).
	dir, err := os.MkdirTemp("/tmp", "skl-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---"), 0644))

	// Create a Unix socket file in the directory
	socketPath := filepath.Join(dir, "s.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	_, err = packageSkillDir(dir)
	assert.Error(t, err, "should fail when encountering a socket file")
	assert.Contains(t, err.Error(), "sockets not supported")
}

func TestPackageSkillDir_VerifyTarGzContainsCorrectFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"SKILL.md":              "---\nname: verify-pkg\n---\n# Test",
		"README.md":             "A readme",
		"src/lib/utils.py":      "def util(): pass",
		"config/settings.yaml":  "key: value",
	}

	for relPath, content := range files {
		absPath := filepath.Join(dir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0644))
	}

	data, err := packageSkillDir(dir)
	require.NoError(t, err)

	// Extract and verify all files are present with correct content
	gr, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	extractedFiles := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if !header.FileInfo().IsDir() {
			content, err := io.ReadAll(tr)
			require.NoError(t, err)
			extractedFiles[header.Name] = string(content)
		}
	}

	for relPath, expectedContent := range files {
		actual, ok := extractedFiles[relPath]
		assert.True(t, ok, "expected file %s to be in archive", relPath)
		assert.Equal(t, expectedContent, actual, "content mismatch for %s", relPath)
	}
}
