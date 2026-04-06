package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSandboxConfig_BothCloneURLs(t *testing.T) {
	req := &ConfigBuildRequest{
		HttpCloneURL: "https://github.com/org/repo.git",
		SshCloneURL:  "git@github.com:org/repo.git",
		SourceBranch: "main",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)

	assert.Equal(t, "https://github.com/org/repo.git", cfg.HttpCloneUrl)
	assert.Equal(t, "git@github.com:org/repo.git", cfg.SshCloneUrl)
	assert.Empty(t, cfg.GetRepositoryUrl(), "deprecated RepositoryUrl must not be populated")
	assert.Equal(t, "main", cfg.SourceBranch)
}

func TestBuildSandboxConfig_OnlyHttpCloneURL(t *testing.T) {
	req := &ConfigBuildRequest{
		HttpCloneURL: "https://github.com/org/repo.git",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)

	assert.Equal(t, "https://github.com/org/repo.git", cfg.HttpCloneUrl)
	assert.Empty(t, cfg.SshCloneUrl)
	assert.Empty(t, cfg.GetRepositoryUrl(), "deprecated RepositoryUrl must not be populated")
}

func TestBuildSandboxConfig_OnlySshCloneURL(t *testing.T) {
	req := &ConfigBuildRequest{
		SshCloneURL: "git@github.com:org/repo.git",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)

	assert.Empty(t, cfg.HttpCloneUrl)
	assert.Equal(t, "git@github.com:org/repo.git", cfg.SshCloneUrl)
	assert.Empty(t, cfg.GetRepositoryUrl(), "deprecated RepositoryUrl must not be populated")
}

func TestBuildSandboxConfig_NilWhenNoURLOrLocalPath(t *testing.T) {
	req := &ConfigBuildRequest{
		PodKey:       "pod-1",
		SourceBranch: "develop",
	}

	cfg := buildSandboxConfig(req)
	assert.Nil(t, cfg, "should return nil when no clone URL or local path is set")
}

func TestBuildSandboxConfig_LocalPathOnly(t *testing.T) {
	req := &ConfigBuildRequest{
		LocalPath: "/home/user/project",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)

	assert.Equal(t, "/home/user/project", cfg.LocalPath)
	assert.Empty(t, cfg.HttpCloneUrl)
	assert.Empty(t, cfg.SshCloneUrl)
	assert.Empty(t, cfg.GetRepositoryUrl(), "deprecated RepositoryUrl must not be populated")
}

func TestBuildSandboxConfig_DefaultPreparationTimeout(t *testing.T) {
	req := &ConfigBuildRequest{
		HttpCloneURL: "https://github.com/org/repo.git",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)
	assert.Equal(t, int32(300), cfg.PreparationTimeout, "default timeout should be 300s")
}

func TestBuildSandboxConfig_CustomPreparationTimeout(t *testing.T) {
	req := &ConfigBuildRequest{
		HttpCloneURL:       "https://github.com/org/repo.git",
		PreparationTimeout: 600,
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)
	assert.Equal(t, int32(600), cfg.PreparationTimeout)
}

func TestBuildSandboxConfig_GitAuthentication(t *testing.T) {
	req := &ConfigBuildRequest{
		HttpCloneURL:   "https://github.com/org/repo.git",
		SshCloneURL:    "git@github.com:org/repo.git",
		CredentialType: "oauth",
		GitToken:       "ghp_test123",
		SSHPrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		TicketSlug:     "PROJ-42",
	}

	cfg := buildSandboxConfig(req)
	require.NotNil(t, cfg)

	assert.Equal(t, "oauth", cfg.CredentialType)
	assert.Equal(t, "ghp_test123", cfg.GitToken)
	assert.Equal(t, "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----", cfg.SshPrivateKey)
	assert.Equal(t, "PROJ-42", cfg.TicketSlug)
	assert.Empty(t, cfg.GetRepositoryUrl(), "deprecated RepositoryUrl must not be populated")
}
