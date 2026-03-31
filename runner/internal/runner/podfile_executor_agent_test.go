package runner

import (
	"encoding/json"
	"strings"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Real production PodFiles from migration 000099.

const realClaudePodFile = `# === Identity ===
AGENT claude
EXECUTABLE claude

# === Mode ===
MODE pty
MODE acp "-p" "--input-format" "stream-json" "--output-format" "stream-json"

# === Configuration ===
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission_mode SELECT("default", "plan", "bypassPermissions") = "default"

# === Environment ===
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_AUTH_TOKEN SECRET OPTIONAL
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL

# === Prompt ===
PROMPT_POSITION prepend

# === Capabilities ===
MCP ON
SKILLS am-delegate, am-channel

# === Build Logic ===
arg "--model" config.model when config.model != ""

if config.permission_mode == "plan" {
  arg "--permission-mode" "plan"
}
if config.permission_mode == "bypassPermissions" {
  arg "--dangerously-skip-permissions"
}

if mcp.enabled {
  plugin_dir = sandbox.root + "/agentsmesh-plugin"

  mkdir plugin_dir
  mkdir plugin_dir + "/.claude-plugin"

  file plugin_dir + "/.claude-plugin/plugin.json" json({
    name: "agentsmesh",
    description: "AgentsMesh collaboration plugin for Claude Code",
    version: "1.0.0"
  })

  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp.servers })

  arg "--plugin-dir" plugin_dir
}
`

const realCodexPodFile = `# === Identity ===
AGENT codex
EXECUTABLE codex

# === Mode ===
MODE pty
MODE acp "app-server"

# === Configuration ===
CONFIG approval_mode SELECT("untrusted", "on-request", "never") = "untrusted"

# === Environment ===
ENV OPENAI_API_KEY SECRET OPTIONAL

# === Prompt ===
PROMPT_POSITION prepend

# === Capabilities ===
MCP ON

# === Build Logic ===
arg "--ask-for-approval" config.approval_mode when config.approval_mode != "" and mode != "acp"

if mcp.enabled {
  mkdir sandbox.work_dir + "/.codex"
  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp.servers })
}
`

const realGeminiPodFile = `# === Identity ===
AGENT gemini
EXECUTABLE gemini

# === Mode ===
MODE pty
MODE acp "--experimental-acp"

# === Configuration ===
CONFIG sandbox_mode BOOL = false

# === Environment ===
ENV GOOGLE_API_KEY SECRET OPTIONAL

# === Prompt ===
PROMPT_POSITION append

# === Capabilities ===
MCP ON FORMAT gemini

# === Build Logic ===
arg "--sandbox" when config.sandbox_mode

if mcp.enabled {
  mkdir sandbox.work_dir + "/.gemini"
  file sandbox.work_dir + "/.gemini/settings.json" json({ mcpServers: mcp.servers })
}
`

const realOpenCodePodFile = `# === Identity ===
AGENT opencode
EXECUTABLE opencode

# === Mode ===
MODE pty
MODE acp "acp"

# === Prompt ===
PROMPT_POSITION prepend

# === Capabilities ===
MCP ON FORMAT opencode

# === Build Logic ===
if mcp.enabled {
  file sandbox.work_dir + "/opencode.json" json({ mcp: mcp.servers })
}
`

func mcpCmd() *runnerv1.CreatePodCommand {
	return &runnerv1.CreatePodCommand{
		PodKey:           "test-pod",
		McpPort:          19000,
		McpBuiltinJson:   `{"agentsmesh":{"type":"http","url":"http://127.0.0.1:19000/mcp"}}`,
		McpInstalledJson: `{"custom":{"command":"node","args":["server.js"]}}`,
	}
}

func TestExecutePodFile_RealClaude_PTY(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realClaudePodFile
	cmd.ConfigValues = map[string]string{"model": "opus", "permission_mode": "bypassPermissions"}
	cmd.Credentials = map[string]string{"ANTHROPIC_API_KEY": "sk-test"}
	cmd.InitialPrompt = "Fix the bug"

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	assert.Equal(t, "claude", result.LaunchCommand)
	assert.Equal(t, "pty", result.Mode)
	assert.Equal(t, "prepend", result.PromptPosition)

	// Prompt prepended as first arg
	require.True(t, len(result.LaunchArgs) > 0)
	assert.Equal(t, "Fix the bug", result.LaunchArgs[0])

	// Config-driven args
	assert.Contains(t, result.LaunchArgs, "--model")
	assert.Contains(t, result.LaunchArgs, "opus")
	assert.Contains(t, result.LaunchArgs, "--dangerously-skip-permissions")
	assert.NotContains(t, result.LaunchArgs, "--permission-mode")

	// ACP mode args NOT in LaunchArgs (we're in PTY mode)
	assert.NotContains(t, result.LaunchArgs, "-p")
	assert.NotContains(t, result.LaunchArgs, "--input-format")

	// MCP plugin dir + files
	assert.Contains(t, result.LaunchArgs, "--plugin-dir")
	assert.Contains(t, result.LaunchArgs, "/tmp/sb/agentsmesh-plugin")

	// Credentials injected
	assert.Equal(t, "sk-test", result.EnvVars["ANTHROPIC_API_KEY"])

	// MCP file contains BOTH builtin and installed servers (merged via mcp.servers)
	mcpJSON := findFileContent(t, result, ".mcp.json")
	assert.Contains(t, mcpJSON, "agentsmesh")
	assert.Contains(t, mcpJSON, "custom")
}

func TestExecutePodFile_RealClaude_ModeArgs(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realClaudePodFile
	cmd.ConfigValues = map[string]string{}

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	// Mode is "pty" (first ModeDecl), ACP args stored but not applied
	assert.Equal(t, "pty", result.Mode)
	assert.NotContains(t, result.LaunchArgs, "-p")
	assert.NotContains(t, result.LaunchArgs, "--output-format")
}

func TestExecutePodFile_RealCodex(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realCodexPodFile
	cmd.ConfigValues = map[string]string{"approval_mode": "on-request"}
	cmd.Credentials = map[string]string{"OPENAI_API_KEY": "sk-openai"}
	cmd.InitialPrompt = "Refactor"

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	assert.Equal(t, "codex", result.LaunchCommand)
	assert.Equal(t, "pty", result.Mode)

	// approval_mode applied (mode != "acp" → condition true)
	assert.Contains(t, result.LaunchArgs, "--ask-for-approval")
	assert.Contains(t, result.LaunchArgs, "on-request")

	// MCP file
	mcpJSON := findFileContent(t, result, "mcp.json")
	assert.Contains(t, mcpJSON, "agentsmesh")
	assert.Contains(t, mcpJSON, "custom")

	assert.Equal(t, "sk-openai", result.EnvVars["OPENAI_API_KEY"])
}

func TestExecutePodFile_RealGemini(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realGeminiPodFile
	cmd.ConfigValues = map[string]string{"sandbox_mode": "false"}
	cmd.Credentials = map[string]string{"GOOGLE_API_KEY": "goog-test"}
	cmd.InitialPrompt = "Hello"

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	assert.Equal(t, "gemini", result.LaunchCommand)
	assert.Equal(t, "append", result.PromptPosition)

	// Prompt appended
	assert.Equal(t, "Hello", result.LaunchArgs[len(result.LaunchArgs)-1])

	// sandbox_mode=false → no --sandbox arg
	assert.NotContains(t, result.LaunchArgs, "--sandbox")

	// MCP FORMAT gemini → httpUrl (not url)
	settingsJSON := findFileContent(t, result, "settings.json")
	assert.Contains(t, settingsJSON, "httpUrl", "gemini format should use httpUrl")
	assert.NotContains(t, settingsJSON, `"url"`)
}

func TestExecutePodFile_RealOpenCode(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realOpenCodePodFile
	cmd.ConfigValues = map[string]string{}
	cmd.InitialPrompt = "Build it"

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	assert.Equal(t, "opencode", result.LaunchCommand)
	assert.Equal(t, "prepend", result.PromptPosition)

	// opencode.json with FORMAT opencode
	ocJSON := findFileContent(t, result, "opencode.json")
	require.NotEmpty(t, ocJSON, "opencode.json should be created")
	// OpenCode format wraps under "mcp" key
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(ocJSON), &parsed))
	_, hasMCP := parsed["mcp"]
	assert.True(t, hasMCP, "opencode.json should have 'mcp' key")
}

func TestExecutePodFile_MCPServersMerge(t *testing.T) {
	// Verify mcp.servers = merged(builtin, installed) with no FORMAT
	podfile := "AGENT test\nMCP ON\nif mcp.enabled {\n  file sandbox.work_dir + \"/mcp.json\" json({ servers: mcp.servers })\n}\n"
	cmd := mcpCmd()
	cmd.PodfileSource = podfile

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	content := findFileContent(t, result, "mcp.json")
	assert.Contains(t, content, "agentsmesh")
	assert.Contains(t, content, "custom")
}

// findFileContent returns the content of the first non-directory file whose path contains substr.
func findFileContent(t *testing.T, result *PodFileResult, substr string) string {
	t.Helper()
	for _, f := range result.FilesToCreate {
		if !f.IsDirectory && strings.Contains(f.Path, substr) {
			return f.Content
		}
	}
	return ""
}
