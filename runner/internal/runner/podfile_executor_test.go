package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testClaudePodFile = `
AGENT claude
EXECUTABLE claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG mcp_enabled BOOL = true
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
MCP ON

arg "--model" config.model when config.model != ""
prompt prepend

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  plugin_dir = sandbox.root + "/agentsmesh-plugin"
  mkdir plugin_dir
  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp_cfg })
  arg "--plugin-dir" plugin_dir
}
`

func TestExecutePodFile_ClaudeCode(t *testing.T) {
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		PodfileSource: testClaudePodFile,
		ConfigValues: map[string]string{
			"model":       "opus",
			"mcp_enabled": "true",
		},
		Credentials:  map[string]string{"ANTHROPIC_API_KEY": "sk-test"},
		IsRunnerHost: false,
		McpPort:      19000,
		McpBuiltinJson: `{"agentsmesh":{"type":"http","url":"http://127.0.0.1:19000/mcp","headers":{"X-Pod-Key":"test-pod"}}}`,
		McpInstalledJson: `{}`,
		InitialPrompt: "Fix the bug",
	}

	result, err := ExecutePodFile(cmd, "/tmp/sandbox", "/tmp/sandbox/workspace")
	require.NoError(t, err)

	assert.Equal(t, "claude", result.LaunchCommand)
	// Prompt prepended
	require.True(t, len(result.LaunchArgs) > 0)
	assert.Equal(t, "Fix the bug", result.LaunchArgs[0])
	assert.Contains(t, result.LaunchArgs, "--model")
	assert.Contains(t, result.LaunchArgs, "opus")
	assert.Contains(t, result.LaunchArgs, "--plugin-dir")
	assert.Contains(t, result.LaunchArgs, "/tmp/sandbox/agentsmesh-plugin")

	// Credentials injected
	assert.Equal(t, "sk-test", result.EnvVars["ANTHROPIC_API_KEY"])

	// Files created (dirs + .mcp.json)
	require.True(t, len(result.FilesToCreate) > 0)
	hasMcpFile := false
	for _, f := range result.FilesToCreate {
		if !f.IsDirectory && f.Content != "" {
			hasMcpFile = true
			assert.Contains(t, f.Content, "agentsmesh")
		}
	}
	assert.True(t, hasMcpFile)
}

func TestExecutePodFile_GeminiCLI(t *testing.T) {
	geminiPodFile := `
AGENT gemini
CONFIG mcp_enabled BOOL = true
MCP ON

prompt append

if mcp.enabled {
  mcp_cfg = mcp_transform(json_merge(mcp.builtin, mcp.installed), "gemini")
  mkdir sandbox.work_dir + "/.gemini"
  file sandbox.work_dir + "/.gemini/settings.json" json({ mcpServers: mcp_cfg })
}
`
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-gemini",
		PodfileSource: geminiPodFile,
		ConfigValues:  map[string]string{"mcp_enabled": "true"},
		McpPort:       19000,
		McpBuiltinJson: `{"agentsmesh":{"type":"http","url":"http://127.0.0.1:19000/mcp"}}`,
		McpInstalledJson: `{}`,
		InitialPrompt: "Hello",
	}

	result, err := ExecutePodFile(cmd, "/tmp/sandbox", "/tmp/sandbox/ws")
	require.NoError(t, err)

	assert.Equal(t, "gemini", result.LaunchCommand)
	// Prompt appended
	assert.Equal(t, "Hello", result.LaunchArgs[len(result.LaunchArgs)-1])

	// Gemini MCP uses httpUrl
	for _, f := range result.FilesToCreate {
		if !f.IsDirectory && f.Content != "" {
			assert.Contains(t, f.Content, "httpUrl")
		}
	}
}

func TestExecutePodFile_Aider(t *testing.T) {
	aiderPodFile := `
AGENT aider
CONFIG model STRING = ""
MCP OFF
arg "--model" config.model when config.model != ""
prompt none
`
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-aider",
		PodfileSource: aiderPodFile,
		ConfigValues:  map[string]string{"model": "gpt-4"},
		InitialPrompt: "Ignored",
	}

	result, err := ExecutePodFile(cmd, "/tmp/sandbox", "/tmp/sandbox/ws")
	require.NoError(t, err)

	assert.Equal(t, "aider", result.LaunchCommand)
	assert.Contains(t, result.LaunchArgs, "--model")
	assert.Contains(t, result.LaunchArgs, "gpt-4")
	// Prompt none — not in args
	assert.NotContains(t, result.LaunchArgs, "Ignored")
	assert.Empty(t, result.FilesToCreate)
}

func TestExecutePodFile_RunnerHost(t *testing.T) {
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-rh",
		PodfileSource: "AGENT test\nENV MY_KEY SECRET\nprompt prepend\n",
		Credentials:   map[string]string{"MY_KEY": "secret-val"},
		IsRunnerHost:  true,
	}

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	// RunnerHost: credentials NOT injected
	_, has := result.EnvVars["MY_KEY"]
	assert.False(t, has)
}

func TestExecutePodFile_ParseError(t *testing.T) {
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-err",
		PodfileSource: "INVALID SYNTAX !!!",
	}
	_, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestExecutePodFile_EvalError(t *testing.T) {
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-err",
		PodfileSource: "AGENT test\nx = undefined_func()\n",
	}
	_, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "eval")
}

func TestParseConfigValue(t *testing.T) {
	assert.Equal(t, true, parseConfigValue("true"))
	assert.Equal(t, false, parseConfigValue("false"))
	assert.Equal(t, float64(42), parseConfigValue("42"))
	assert.Equal(t, "hello", parseConfigValue("hello"))
	assert.Equal(t, "opus", parseConfigValue("opus"))
}

func TestParseJSON(t *testing.T) {
	m := parseJSON(`{"key":"val"}`)
	assert.Equal(t, "val", m["key"])

	empty := parseJSON("")
	assert.Empty(t, empty)

	invalid := parseJSON("not json")
	assert.Empty(t, invalid)
}
