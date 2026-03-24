package runner

import (
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareAgentHome_NoCodexHome(t *testing.T) {
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:  "test-pod",
			EnvVars: map[string]string{"FOO": "bar"},
		},
	}
	err := builder.prepareAgentHome("/sandbox", "/workspace")
	assert.NoError(t, err)
}

func TestPrepareAgentHome_NilEnvVars(t *testing.T) {
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{PodKey: "test-pod"},
	}
	err := builder.prepareAgentHome("/sandbox", "/workspace")
	assert.NoError(t, err)
}

func TestPrepareAgentHome_CreatesEmptyDir(t *testing.T) {
	sandboxRoot := t.TempDir()
	codexHome := filepath.Join(sandboxRoot, "codex-home")

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:  "test-pod",
			EnvVars: map[string]string{"CODEX_HOME": codexHome},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, dirExists(codexHome))
}

func TestPrepareAgentHome_ResolvesTemplateVars(t *testing.T) {
	sandboxRoot := t.TempDir()

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:  "test-pod",
			EnvVars: map[string]string{"CODEX_HOME": "{{.sandbox.root_path}}/codex-home"},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)

	codexHome := filepath.Join(sandboxRoot, "codex-home")
	assert.True(t, dirExists(codexHome))
}

func TestPrepareAgentHome_CopiesUserConfig(t *testing.T) {
	sandboxRoot := t.TempDir()
	codexHome := filepath.Join(sandboxRoot, "codex-home")

	// Create a fake user ~/.codex/ with config.toml
	userHome := t.TempDir()
	userCodexDir := filepath.Join(userHome, ".codex")
	require.NoError(t, os.MkdirAll(userCodexDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(userCodexDir, "config.toml"),
		[]byte("[mcp_servers.user_server]\ncommand = \"my-server\"\n"),
		0644,
	))
	// Also create a credentials file that should be preserved
	require.NoError(t, os.WriteFile(
		filepath.Join(userCodexDir, "credentials.json"),
		[]byte(`{"token":"secret"}`),
		0644,
	))

	// Test copyDirSelective and mergeTomlMcpServers directly
	// (prepareAgentHome uses os.UserHomeDir() which can't be easily overridden)

	// Test copyDirSelective
	err := copyDirSelective(userCodexDir, codexHome)
	require.NoError(t, err)

	// Verify config.toml was copied
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "user_server")

	// Verify credentials were copied
	data, err = os.ReadFile(filepath.Join(codexHome, "credentials.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "secret")
}

func TestCopyDirSelective_SkipsSessions(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create sessions/ directory (should be skipped)
	sessionsDir := filepath.Join(src, "sessions", "2026", "03")
	require.NoError(t, os.MkdirAll(sessionsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sessionsDir, "log.jsonl"), []byte("data"), 0644))

	// Create config.toml (should be copied)
	require.NoError(t, os.WriteFile(filepath.Join(src, "config.toml"), []byte("key = \"val\""), 0644))

	err := copyDirSelective(src, dst)
	require.NoError(t, err)

	// config.toml should exist
	assert.FileExists(t, filepath.Join(dst, "config.toml"))
	// sessions/ should NOT exist
	_, err = os.Stat(filepath.Join(dst, "sessions"))
	assert.True(t, os.IsNotExist(err), "sessions/ should be skipped")
}

func TestMergeTomlMcpServers_NoExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	platformContent := "[mcp_servers.agentsmesh]\nurl = \"http://localhost:19000/mcp\"\n"

	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	// File should be created with platform content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "agentsmesh")
	assert.Contains(t, string(data), "localhost:19000")
}

func TestMergeTomlMcpServers_PreservesUserConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Write existing user config with their own settings and MCP server
	existingContent := `model = "gpt-4"
send_logs = false

[mcp_servers.user_github]
command = "gh-mcp"
args = ["serve"]
`
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0644))

	// Merge platform MCP servers
	platformContent := "[mcp_servers.agentsmesh]\nurl = \"http://localhost:19000/mcp\"\n"

	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

	// User's model and settings should be preserved
	assert.Contains(t, content, "gpt-4")
	// User's MCP server should be preserved
	assert.Contains(t, content, "user_github")
	assert.Contains(t, content, "gh-mcp")
	// Platform MCP server should be added
	assert.Contains(t, content, "agentsmesh")
	assert.Contains(t, content, "localhost:19000")
}

func TestMergeTomlMcpServers_OverridesSameKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	// Write existing config with an agentsmesh entry
	existingContent := `[mcp_servers.agentsmesh]
url = "http://old-server:9000/mcp"
`
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0644))

	// Merge with new agentsmesh config
	platformContent := `[mcp_servers.agentsmesh]
url = "http://localhost:19000/mcp"
`
	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

	// New URL should override old
	assert.Contains(t, content, "localhost:19000")
	assert.NotContains(t, content, "old-server:9000")
}

func TestMergeTomlMcpServers_EmptyPlatformContent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	existingContent := "model = \"gpt-4\"\n"
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0644))

	err := mergeTomlMcpServers(configPath, "")
	require.NoError(t, err)

	// File should be unchanged
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(data))
}
