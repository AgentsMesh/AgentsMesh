package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeTomlMcpServers_NoExistingConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	platformContent := "[mcp_servers.agentsmesh]\nurl = \"http://localhost:19000/mcp\"\n"

	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "agentsmesh")
	assert.Contains(t, string(data), "localhost:19000")
}

func TestMergeTomlMcpServers_PreservesUserConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	existingContent := `model = "gpt-4"
send_logs = false

[mcp_servers.user_github]
command = "gh-mcp"
args = ["serve"]
`
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0644))

	platformContent := "[mcp_servers.agentsmesh]\nurl = \"http://localhost:19000/mcp\"\n"

	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "gpt-4")
	assert.Contains(t, content, "user_github")
	assert.Contains(t, content, "gh-mcp")
	assert.Contains(t, content, "agentsmesh")
	assert.Contains(t, content, "localhost:19000")
}

func TestMergeTomlMcpServers_OverridesSameKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	existingContent := `[mcp_servers.agentsmesh]
url = "http://old-server:9000/mcp"
`
	require.NoError(t, os.WriteFile(configPath, []byte(existingContent), 0644))

	platformContent := `[mcp_servers.agentsmesh]
url = "http://localhost:19000/mcp"
`
	err := mergeTomlMcpServers(configPath, platformContent)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)

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

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(data))
}
