package extension

import (
	"context"
	"errors"
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
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       extension.AuthTypeNone,
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

	source := &extension.SkillRegistry{
		ID:             1,
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "main",
		AuthType:       "",
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

	imp.SetCredentialDecryptor(func(encrypted string) (string, error) {
		return "decrypted-" + encrypted, nil
	})

	authCloneCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		authCloneCalled = true
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

	imp.SetCredentialDecryptor(func(encrypted string) (string, error) {
		return "", errors.New("decryption failed: corrupted data")
	})

	authCloneCalled := false
	imp.gitCloneAuthFn = func(_ context.Context, url, branch, targetDir, authType, credential string) error {
		authCloneCalled = true
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
