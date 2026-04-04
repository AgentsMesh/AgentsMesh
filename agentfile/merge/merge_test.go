package merge

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, src string) *parser.Program {
	t.Helper()
	prog, errs := parser.Parse(src)
	require.Empty(t, errs, "parse errors: %v", errs)
	return prog
}

func TestMerge_ConfigOverride(t *testing.T) {
	base := parse(t, `
AGENT claude
CONFIG model SELECT("", "sonnet", "opus") = "sonnet"
CONFIG permission SELECT("default", "plan") = "default"

arg "--model" config.model when config.model != ""
`)
	slice := parse(t, `
CONFIG model = "opus"
CONFIG permission = "plan"
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	// Config defaults overridden
	assert.Equal(t, "opus", spec.Config[0].Default)
	assert.Equal(t, "plan", spec.Config[1].Default)
	// Agent preserved from base
	assert.Equal(t, "claude", spec.Agent.Command)
}

func TestMerge_EnvOverride(t *testing.T) {
	base := parse(t, `
AGENT test
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
`)
	slice := parse(t, `
ENV ANTHROPIC_API_KEY SECRET
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	require.Len(t, spec.Env, 1)
	assert.Equal(t, "secret", spec.Env[0].Source)
	assert.False(t, spec.Env[0].Optional) // OPTIONAL removed
}

func TestMerge_SkillsUnion(t *testing.T) {
	base := parse(t, `
AGENT test
SKILLS am-delegate, am-channel
`)
	slice := parse(t, `
SKILLS custom-skill
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	assert.Equal(t, []string{"am-delegate", "am-channel", "custom-skill"}, spec.Skills)
}

func TestMerge_SkillsNoDuplicate(t *testing.T) {
	base := parse(t, `
AGENT test
SKILLS am-delegate, am-channel
`)
	slice := parse(t, `
SKILLS am-delegate, new-skill
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	assert.Equal(t, []string{"am-delegate", "am-channel", "new-skill"}, spec.Skills)
}

func TestMerge_RepoOverride(t *testing.T) {
	base := parse(t, `AGENT test`)
	slice := parse(t, `
REPO "https://github.com/org/project"
BRANCH "main"
GIT_CREDENTIAL oauth
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	require.NotNil(t, spec.Repo)
	assert.Equal(t, "https://github.com/org/project", spec.Repo.URL)
	assert.Equal(t, "main", spec.Repo.Branch)
}

func TestMerge_MCPToggle(t *testing.T) {
	base := parse(t, `
AGENT test
MCP ON
`)
	slice := parse(t, `MCP OFF`)
	Merge(base, slice)
	spec := extract.Extract(base)

	require.NotNil(t, spec.MCP)
	assert.False(t, spec.MCP.Enabled)
}

func TestMerge_StatementsAppended(t *testing.T) {
	base := parse(t, `
AGENT test
arg "--base-flag"
`)
	slice := parse(t, `
arg "--slice-flag"
`)
	Merge(base, slice)

	assert.Len(t, base.Statements, 2)
}

func TestMerge_RemoveDecl_Env(t *testing.T) {
	base := parse(t, `
AGENT test
ENV ANTHROPIC_API_KEY SECRET
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
`)
	slice := parse(t, `
REMOVE ENV ANTHROPIC_BASE_URL
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	// Only API_KEY remains
	require.Len(t, spec.Env, 1)
	assert.Equal(t, "ANTHROPIC_API_KEY", spec.Env[0].Name)
}

func TestMerge_RemoveDecl_Skills(t *testing.T) {
	base := parse(t, `
AGENT test
SKILLS am-delegate, am-channel
`)
	// REMOVE SKILLS works at eval level, not extract level.
	// At merge level, we test that RemoveDecl passes through.
	slice := parse(t, `
REMOVE SKILLS am-delegate
`)
	Merge(base, slice)

	// Run eval to verify remove works end-to-end
	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Equal(t, []string{"am-channel"}, ctx.Result.Skills)
}

func TestMerge_RemoveDecl_Arg(t *testing.T) {
	base := parse(t, `
AGENT test
arg "--verbose"
arg "--model" "opus"
`)
	slice := parse(t, `
REMOVE arg "--verbose"
`)
	Merge(base, slice)

	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Equal(t, []string{"--model", "opus"}, ctx.Result.LaunchArgs)
}

func TestMerge_RemoveDecl_ArgWithValue(t *testing.T) {
	base := parse(t, `
AGENT test
arg "--model" "sonnet"
arg "--permission-mode" "plan"
`)
	slice := parse(t, `
REMOVE arg "--model"
`)
	Merge(base, slice)

	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	// --model and its value "sonnet" both removed
	assert.Equal(t, []string{"--permission-mode", "plan"}, ctx.Result.LaunchArgs)
}

func TestMerge_Recursive(t *testing.T) {
	base := parse(t, `
AGENT claude
CONFIG model = "sonnet"
SKILLS am-delegate
arg "--base"
`)
	layer1 := parse(t, `
CONFIG model = "opus"
SKILLS am-channel
arg "--layer1"
`)
	layer2 := parse(t, `
REPO "https://github.com/org/repo"
arg "--layer2"
`)
	// Recursive merge: (base + layer1) + layer2
	Merge(base, layer1)
	Merge(base, layer2)
	spec := extract.Extract(base)

	assert.Equal(t, "claude", spec.Agent.Command)
	assert.Equal(t, "opus", spec.Config[0].Default)        // layer1 override
	assert.Equal(t, []string{"am-delegate", "am-channel"}, spec.Skills) // union
	require.NotNil(t, spec.Repo)
	assert.Equal(t, "https://github.com/org/repo", spec.Repo.URL) // layer2

	assert.Len(t, base.Statements, 3) // base + layer1 + layer2
}

func TestMerge_FullE2E(t *testing.T) {
	base := parse(t, `
AGENT claude
EXECUTABLE claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG mcp_enabled BOOL = true
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
MCP ON
SKILLS am-delegate, am-channel

arg "--model" config.model when config.model != ""
PROMPT_POSITION prepend
`)
	userSlice := parse(t, `
CONFIG model = "opus"
REPO "https://github.com/org/project"
BRANCH "main"
`)
	Merge(base, userSlice)

	// Eval with config from merged declarations
	ctx := eval.NewContext(map[string]interface{}{
		"config": map[string]interface{}{
			"model":       "opus",
			"mcp_enabled": true,
		},
		"mcp": map[string]interface{}{
			"enabled": true,
		},
	})
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Equal(t, "claude", ctx.Result.LaunchCommand)
	assert.Contains(t, ctx.Result.LaunchArgs, "--model")
	assert.Contains(t, ctx.Result.LaunchArgs, "opus")
	assert.Equal(t, "prepend", ctx.Result.PromptPosition)
	assert.Equal(t, "https://github.com/org/project", ctx.Result.Sandbox.RepoURL)
	assert.Equal(t, "main", ctx.Result.Sandbox.Branch)
}
