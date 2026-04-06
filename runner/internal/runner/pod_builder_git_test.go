package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkspace implements workspace.WorkspaceManagerInterface for testing.
type mockWorkspace struct {
	result *workspace.WorktreeResult
	err    error
	opts   []workspace.WorktreeOption // captured options
}

func (m *mockWorkspace) CreateWorktree(_ context.Context, _, _, _ string) (*workspace.WorktreeResult, error) {
	return m.result, m.err
}
func (m *mockWorkspace) CreateWorktreeWithOptions(_ context.Context, _, _, _ string, opts ...workspace.WorktreeOption) (*workspace.WorktreeResult, error) {
	m.opts = opts
	return m.result, m.err
}
func (m *mockWorkspace) RemoveWorktree(_ context.Context, _ string) error { return nil }
func (m *mockWorkspace) CleanupOldWorktrees(_ context.Context) error      { return nil }
func (m *mockWorkspace) TempWorkspace(_ string) string                    { return "" }
func (m *mockWorkspace) GetWorkspaceRoot() string                         { return "" }
func (m *mockWorkspace) ListWorktrees() ([]string, error)                 { return nil, nil }

func gitBuilder(ws workspace.WorkspaceManagerInterface, cfg *runnerv1.SandboxConfig) *PodBuilder {
	r := &Runner{cfg: &config.Config{WorkspaceRoot: os.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "git-test-pod",
		AgentfileSource: "AGENT echo\n",
		SandboxConfig: cfg,
	}
	return NewPodBuilder(PodBuilderDeps{Config: r.cfg, Workspace: ws}).WithCommand(cmd)
}

func TestSetupGitWorktree_Success(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		SourceBranch:   "main",
		CredentialType: "runner_local",
	})
	path, branch, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/ws", path)
	assert.Equal(t, "main", branch)
}

func TestSetupGitWorktree_EmptyURL(t *testing.T) {
	ws := &mockWorkspace{}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{CredentialType: "runner_local"})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.Error(t, err)
	var podErr *client.PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, client.ErrCodeGitClone, podErr.Code)
}

func TestSetupGitWorktree_NilWorkspace(t *testing.T) {
	b := gitBuilder(nil, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		CredentialType: "runner_local",
	})
	// Set Workspace to nil explicitly
	b.deps.Workspace = nil
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.Error(t, err)
	var podErr *client.PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, client.ErrCodeGitWorktree, podErr.Code)
}

func TestSetupGitWorktree_OAuthToken(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "dev"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		SourceBranch:   "dev",
		CredentialType: "oauth",
		GitToken:       "gho_xxx",
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.NoError(t, err)
	// Verify git token option was passed
	assert.NotEmpty(t, ws.opts, "should pass WithGitToken option")
}

func TestSetupGitWorktree_PATToken(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		CredentialType: "pat",
		GitToken:       "ghp_xxx",
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.NoError(t, err)
	assert.NotEmpty(t, ws.opts)
}

func TestSetupGitWorktree_SSHKey(t *testing.T) {
	sandbox := t.TempDir()
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: sandbox + "/workspace", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		SshCloneUrl:    "git@github.com:org/repo.git",
		CredentialType: "ssh_key",
		SshPrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
	})
	_, _, err := b.setupGitWorktree(context.Background(), sandbox, b.cmd.SandboxConfig)
	require.NoError(t, err)
	// Verify SSH key was written
	keyFile := filepath.Join(sandbox, ".ssh_key")
	data, readErr := os.ReadFile(keyFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "OPENSSH PRIVATE KEY")
	// Verify permissions (Unix only — Windows uses ACLs)
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(keyFile)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestSetupGitWorktree_HttpCloneURL_Preferred(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://new-url.com/repo.git",
		SshCloneUrl:    "git@github.com:org/repo.git",
		CredentialType: "runner_local",
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.NoError(t, err)
	// HTTP clone URL should be used when both available
	assert.True(t, len(ws.opts) >= 2, "should pass both HttpCloneURL and SshCloneURL options")
}

func TestSetupGitWorktree_AuthError(t *testing.T) {
	ws := &mockWorkspace{err: fmt.Errorf("authentication failed: Permission denied")}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		CredentialType: "runner_local",
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.Error(t, err)
	var podErr *client.PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, client.ErrCodeGitAuth, podErr.Code)
}

func TestSetupGitWorktree_CloneError(t *testing.T) {
	ws := &mockWorkspace{err: fmt.Errorf("failed to clone repository")}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		CredentialType: "runner_local",
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.Error(t, err)
	var podErr *client.PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, client.ErrCodeGitClone, podErr.Code)
}

// TestSetupGitWorktree_DeprecatedRepositoryUrl_Ignored is a regression test
// ensuring the deprecated RepositoryUrl proto field is never used as a fallback
// when HttpCloneUrl and SshCloneUrl are both empty. The builder must return
// ErrCodeGitClone instead of silently cloning from the legacy field.
func TestSetupGitWorktree_DeprecatedRepositoryUrl_Ignored(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		RepositoryUrl:  "https://github.com/org/repo.git", //nolint:staticcheck // testing deprecated field
		CredentialType: "runner_local",
		// HttpCloneUrl and SshCloneUrl intentionally left empty
	})
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.Error(t, err, "should not fall back to deprecated RepositoryUrl")
	var podErr *client.PodError
	require.ErrorAs(t, err, &podErr)
	assert.Equal(t, client.ErrCodeGitClone, podErr.Code)
	assert.Contains(t, podErr.Message, "http_clone_url or ssh_clone_url is required")
}

func TestSetupGitWorktree_UnknownCredentialType(t *testing.T) {
	ws := &mockWorkspace{result: &workspace.WorktreeResult{Path: "/tmp/ws", Branch: "main"}}
	b := gitBuilder(ws, &runnerv1.SandboxConfig{
		HttpCloneUrl:   "https://github.com/org/repo.git",
		CredentialType: "unknown_type",
	})
	// Unknown type falls back to runner_local (no error, no token option)
	_, _, err := b.setupGitWorktree(context.Background(), t.TempDir(), b.cmd.SandboxConfig)
	require.NoError(t, err)
}
