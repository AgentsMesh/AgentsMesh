package extract

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtract_ClaudeCode(t *testing.T) {
	input := `
AGENT claude
EXECUTABLE claude

CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission SELECT("default", "plan", "bypass") = "default"
CONFIG mcp_enabled BOOL = true

ENV ANTHROPIC_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
ENV TERM = "xterm-256color"

MCP ON
SKILLS am-delegate, am-channel

# Build logic is ignored by extractor
arg "--model" config.model when config.model != ""
PROMPT_POSITION prepend
`
	prog, errs := parser.Parse(input)
	require.Empty(t, errs)

	spec := Extract(prog)

	// Agent
	assert.Equal(t, "claude", spec.Agent.Command)
	assert.Equal(t, "claude", spec.Agent.Executable)

	// Config
	require.Len(t, spec.Config, 3)
	assert.Equal(t, "model", spec.Config[0].Name)
	assert.Equal(t, "select", spec.Config[0].Type)
	assert.Equal(t, []string{"", "sonnet", "opus"}, spec.Config[0].Options)
	assert.Equal(t, "", spec.Config[0].Default)
	assert.Equal(t, "mcp_enabled", spec.Config[2].Name)
	assert.Equal(t, "boolean", spec.Config[2].Type)
	assert.Equal(t, true, spec.Config[2].Default)

	// Env
	require.Len(t, spec.Env, 3)
	assert.Equal(t, "ANTHROPIC_API_KEY", spec.Env[0].Name)
	assert.Equal(t, "secret", spec.Env[0].Source)
	assert.True(t, spec.Env[0].Optional)
	assert.Equal(t, "ANTHROPIC_BASE_URL", spec.Env[1].Name)
	assert.Equal(t, "text", spec.Env[1].Source)
	assert.Equal(t, "TERM", spec.Env[2].Name)
	assert.Equal(t, "", spec.Env[2].Source)
	assert.Equal(t, "xterm-256color", spec.Env[2].Value)

	// MCP
	require.NotNil(t, spec.MCP)
	assert.True(t, spec.MCP.Enabled)

	// Skills
	assert.Equal(t, []string{"am-delegate", "am-channel"}, spec.Skills)

	// No repo declared
	assert.Nil(t, spec.Repo)
	// No setup declared
	assert.Nil(t, spec.Setup)
}

func TestExtract_WithRepo(t *testing.T) {
	input := `
AGENT gemini
REPO "https://github.com/org/project"
BRANCH "main"
GIT_CREDENTIAL oauth
`
	prog, errs := parser.Parse(input)
	require.Empty(t, errs)

	spec := Extract(prog)

	assert.Equal(t, "gemini", spec.Agent.Command)
	require.NotNil(t, spec.Repo)
	assert.Equal(t, "https://github.com/org/project", spec.Repo.URL)
	assert.Equal(t, "main", spec.Repo.Branch)
	assert.Equal(t, "oauth", spec.Repo.CredentialType)
}

func TestExtract_WithSetup(t *testing.T) {
	input := `
AGENT claude
SETUP timeout=120 <<EOF
npm install
npm run build
EOF
`
	prog, errs := parser.Parse(input)
	require.Empty(t, errs)

	spec := Extract(prog)

	require.NotNil(t, spec.Setup)
	assert.Equal(t, 120, spec.Setup.Timeout)
	assert.Contains(t, spec.Setup.Script, "npm install")
	assert.Contains(t, spec.Setup.Script, "npm run build")
}

func TestExtract_Minimal(t *testing.T) {
	input := `AGENT aider`
	prog, errs := parser.Parse(input)
	require.Empty(t, errs)

	spec := Extract(prog)

	assert.Equal(t, "aider", spec.Agent.Command)
	assert.Empty(t, spec.Config)
	assert.Empty(t, spec.Env)
	assert.Nil(t, spec.Repo)
	assert.Nil(t, spec.MCP)
	assert.Empty(t, spec.Skills)
	assert.Nil(t, spec.Setup)
}

func TestExtract_MCPOff(t *testing.T) {
	input := `
AGENT aider
MCP OFF
`
	prog, errs := parser.Parse(input)
	require.Empty(t, errs)

	spec := Extract(prog)

	require.NotNil(t, spec.MCP)
	assert.False(t, spec.MCP.Enabled)
}
