package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Declarations(t *testing.T) {
	input := `
AGENT claude
EXECUTABLE claude

CONFIG model SELECT("", "sonnet", "opus") = "sonnet"
CONFIG mcp_enabled BOOL = true
CONFIG temperature NUMBER = 0.7

ENV ANTHROPIC_API_KEY SECRET
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
ENV TERM = "xterm-256color"

REPO "https://github.com/org/project"
BRANCH "main"
GIT_CREDENTIAL oauth

MCP ON
SKILLS am-delegate, am-channel
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)

	assert.Len(t, prog.Declarations, 13)

	// AGENT
	agent := prog.Declarations[0].(*AgentDecl)
	assert.Equal(t, "claude", agent.Command)

	// EXECUTABLE
	exec := prog.Declarations[1].(*ExecutableDecl)
	assert.Equal(t, "claude", exec.Name)

	// CONFIG model
	cfg := prog.Declarations[2].(*ConfigDecl)
	assert.Equal(t, "model", cfg.Name)
	assert.Equal(t, "select", cfg.TypeName)
	assert.Equal(t, []string{"", "sonnet", "opus"}, cfg.Options)
	assert.Equal(t, "sonnet", cfg.Default)

	// CONFIG mcp_enabled
	cfg2 := prog.Declarations[3].(*ConfigDecl)
	assert.Equal(t, "mcp_enabled", cfg2.Name)
	assert.Equal(t, "boolean", cfg2.TypeName)
	assert.Equal(t, true, cfg2.Default)

	// CONFIG temperature
	cfg3 := prog.Declarations[4].(*ConfigDecl)
	assert.Equal(t, "temperature", cfg3.Name)
	assert.Equal(t, "number", cfg3.TypeName)
	assert.Equal(t, 0.7, cfg3.Default)

	// ENV SECRET
	env1 := prog.Declarations[5].(*EnvDecl)
	assert.Equal(t, "ANTHROPIC_API_KEY", env1.Name)
	assert.Equal(t, "secret", env1.Source)
	assert.False(t, env1.Optional)

	// ENV TEXT OPTIONAL
	env2 := prog.Declarations[6].(*EnvDecl)
	assert.Equal(t, "ANTHROPIC_BASE_URL", env2.Name)
	assert.Equal(t, "text", env2.Source)
	assert.True(t, env2.Optional)

	// ENV fixed value
	env3 := prog.Declarations[7].(*EnvDecl)
	assert.Equal(t, "TERM", env3.Name)
	assert.Equal(t, "", env3.Source)
	assert.Equal(t, "xterm-256color", env3.Value)

	// REPO
	repo := prog.Declarations[8].(*RepoDecl)
	assert.IsType(t, &StringLit{}, repo.Value)

	// BRANCH
	branch := prog.Declarations[9].(*BranchDecl)
	assert.IsType(t, &StringLit{}, branch.Value)

	// GIT_CREDENTIAL
	gitCred := prog.Declarations[10].(*GitCredentialDecl)
	assert.Equal(t, "oauth", gitCred.Type)

	// MCP
	mcp := prog.Declarations[11].(*McpDecl)
	assert.True(t, mcp.Enabled)

	// SKILLS
	skills := prog.Declarations[12].(*SkillsDecl)
	assert.Equal(t, []string{"am-delegate", "am-channel"}, skills.Slugs)
}

func TestParse_BuildStatements(t *testing.T) {
	input := `
arg "--model" config.model when config.model != ""
arg "--flag"
PROMPT_POSITION prepend
mkdir sandbox.root + "/plugin"
x = json_merge(mcp.builtin, mcp.installed)
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)

	// PROMPT_POSITION is a declaration
	require.Len(t, prog.Declarations, 1)
	promptDecl := prog.Declarations[0].(*PromptPositionDecl)
	assert.Equal(t, "prepend", promptDecl.Mode)

	// Remaining are statements
	assert.Len(t, prog.Statements, 4)

	// arg with when
	argStmt := prog.Statements[0].(*ArgStmt)
	assert.Len(t, argStmt.Args, 2)
	assert.NotNil(t, argStmt.When)

	// arg without when
	argStmt2 := prog.Statements[1].(*ArgStmt)
	assert.Len(t, argStmt2.Args, 1)
	assert.Nil(t, argStmt2.When)

	// mkdir with + expression
	mkdirStmt := prog.Statements[2].(*MkdirStmt)
	binExpr, ok := mkdirStmt.Path.(*BinaryExpr)
	require.True(t, ok)
	assert.Equal(t, "+", binExpr.Op)

	// assignment with function call
	assign := prog.Statements[3].(*AssignStmt)
	assert.Equal(t, "x", assign.Name)
	call, ok := assign.Value.(*CallExpr)
	require.True(t, ok)
	assert.Equal(t, "json_merge", call.Func)
	assert.Len(t, call.Args, 2)
}

func TestParse_IfStatement(t *testing.T) {
	input := `
if config.permission == "plan" {
  arg "--permission-mode" "plan"
} else {
  arg "--default"
}
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)
	require.Len(t, prog.Statements, 1)

	ifStmt := prog.Statements[0].(*IfStmt)

	cond := ifStmt.Condition.(*BinaryExpr)
	assert.Equal(t, "==", cond.Op)

	assert.Len(t, ifStmt.Body, 1)
	assert.Len(t, ifStmt.Else, 1)
}

func TestParse_NestedIf(t *testing.T) {
	input := `
if mcp.enabled {
  if config.format == "claude" {
    arg "--mcp-config" "path"
  }
}
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)
	require.Len(t, prog.Statements, 1)

	outer := prog.Statements[0].(*IfStmt)
	require.Len(t, outer.Body, 1)

	inner := outer.Body[0].(*IfStmt)
	assert.Len(t, inner.Body, 1)
}

func TestParse_ForStatement(t *testing.T) {
	input := `
for name, server in servers {
  arg "-c" name + "=" + server.url
}
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)
	require.Len(t, prog.Statements, 1)

	forStmt := prog.Statements[0].(*ForStmt)
	assert.Equal(t, "name", forStmt.Key)
	assert.Equal(t, "server", forStmt.Value)
	assert.Len(t, forStmt.Body, 1)
}

func TestParse_ObjectLiteral(t *testing.T) {
	input := `
x = json({
  name: "agentsmesh",
  version: "1.0.0",
  enabled: true
})
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)
	require.Len(t, prog.Statements, 1)

	assign := prog.Statements[0].(*AssignStmt)
	call := assign.Value.(*CallExpr)
	assert.Equal(t, "json", call.Func)
	require.Len(t, call.Args, 1)

	obj := call.Args[0].(*ObjectLit)
	assert.Len(t, obj.Fields, 3)
	assert.Equal(t, "name", obj.Fields[0].Key)
	assert.Equal(t, "version", obj.Fields[1].Key)
	assert.Equal(t, "enabled", obj.Fields[2].Key)
}

func TestParse_FileWithHeredoc(t *testing.T) {
	input := `
file sandbox.root + "/config.json" <<EOF
{
  "key": "value"
}
EOF
`
	prog, errs := Parse(input)
	require.Empty(t, errs, "parse errors: %v", errs)
	require.Len(t, prog.Statements, 1)

	fileStmt := prog.Statements[0].(*FileStmt)
	_, ok := fileStmt.Path.(*BinaryExpr)
	assert.True(t, ok)

	heredoc := fileStmt.Content.(*HeredocLit)
	assert.Contains(t, heredoc.Content, `"key": "value"`)
}
