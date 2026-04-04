package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const claudeCodeAgentFile = `# Claude Code AgentFile
AGENT claude
EXECUTABLE claude

CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission SELECT("default", "plan", "bypass") = "default"
CONFIG mcp_enabled BOOL = true

ENV ANTHROPIC_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_AUTH_TOKEN SECRET OPTIONAL
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL

MCP ON
SKILLS am-delegate, am-channel

PROMPT_POSITION prepend

# --- Build logic ---

arg "--model" config.model when config.model != ""

if config.permission == "plan" {
  arg "--permission-mode" "plan"
}
if config.permission == "bypass" {
  arg "--dangerously-skip-permissions"
}

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  plugin_dir = sandbox.root + "/agentsmesh-plugin"

  mkdir plugin_dir
  mkdir plugin_dir + "/.claude-plugin"

  file plugin_dir + "/.claude-plugin/plugin.json" json({
    name: "agentsmesh",
    description: "AgentsMesh collaboration plugin",
    version: "1.0.0"
  })

  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp_cfg })

  arg "--plugin-dir" plugin_dir
}
`

func TestParse_FullClaudeCode(t *testing.T) {
	prog, errs := Parse(claudeCodeAgentFile)
	require.Empty(t, errs, "parse errors: %v", errs)

	// Declarations: AGENT, EXECUTABLE, 3x CONFIG, 3x ENV, MCP, SKILLS, PROMPT_POSITION = 11
	assert.Len(t, prog.Declarations, 11)

	assert.IsType(t, &AgentDecl{}, prog.Declarations[0])
	assert.IsType(t, &ExecutableDecl{}, prog.Declarations[1])
	assert.IsType(t, &ConfigDecl{}, prog.Declarations[2])
	assert.IsType(t, &ConfigDecl{}, prog.Declarations[3])
	assert.IsType(t, &ConfigDecl{}, prog.Declarations[4])
	assert.IsType(t, &EnvDecl{}, prog.Declarations[5])
	assert.IsType(t, &EnvDecl{}, prog.Declarations[6])
	assert.IsType(t, &EnvDecl{}, prog.Declarations[7])
	assert.IsType(t, &McpDecl{}, prog.Declarations[8])
	assert.IsType(t, &SkillsDecl{}, prog.Declarations[9])
	promptPos := prog.Declarations[10].(*PromptPositionDecl)
	assert.Equal(t, "prepend", promptPos.Mode)

	// Statements: arg, if, if, if(mcp block) = 4
	assert.Len(t, prog.Statements, 4)

	// First statement: arg with when
	argStmt := prog.Statements[0].(*ArgStmt)
	assert.NotNil(t, argStmt.When)

	// Second: if config.permission == "plan"
	ifPlan := prog.Statements[1].(*IfStmt)
	assert.Len(t, ifPlan.Body, 1)

	// Third: if config.permission == "bypass"
	ifBypass := prog.Statements[2].(*IfStmt)
	assert.Len(t, ifBypass.Body, 1)

	// Fourth: if mcp.enabled { ... }
	ifMcp := prog.Statements[3].(*IfStmt)
	// Body should have: assign, assign, mkdir, mkdir, file, file, arg
	assert.Len(t, ifMcp.Body, 7)
}
