package serialize

import (
	"testing"

	"github.com/anthropics/agentsmesh/podfile/eval"
	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/merge"
	"github.com/anthropics/agentsmesh/podfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSerialize_ConfigOverride(t *testing.T) {
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
	merged := merge.Merge(base, slice)
	src := Serialize(merged)

	// Re-parse the serialized result and extract
	reparsed := parse(t, src)
	spec := extract.Extract(reparsed)
	assert.Equal(t, "claude", spec.Agent.Command)
	assert.Equal(t, "opus", spec.Config[0].Default)
	assert.Equal(t, "plan", spec.Config[1].Default)
}

func TestMergeSerialize_SkillsUnion(t *testing.T) {
	base := parse(t, `
AGENT test
SKILLS am-delegate, am-channel
`)
	slice := parse(t, `SKILLS custom-skill`)
	merged := merge.Merge(base, slice)
	src := Serialize(merged)

	reparsed := parse(t, src)
	spec := extract.Extract(reparsed)
	assert.Equal(t, []string{"am-delegate", "am-channel", "custom-skill"}, spec.Skills)
}

func TestMergeSerialize_RemoveDecl(t *testing.T) {
	base := parse(t, `
AGENT test
ENV API_KEY SECRET
ENV BASE_URL TEXT OPTIONAL
`)
	slice := parse(t, `REMOVE ENV BASE_URL`)
	merged := merge.Merge(base, slice)
	src := Serialize(merged)

	reparsed := parse(t, src)
	spec := extract.Extract(reparsed)
	require.Len(t, spec.Env, 1)
	assert.Equal(t, "API_KEY", spec.Env[0].Name)
}

func TestMergeSerialize_FullE2E(t *testing.T) {
	base := parse(t, `
AGENT claude
EXECUTABLE claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG mcp_enabled BOOL = true
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
MCP ON
SKILLS am-delegate, am-channel

arg "--model" config.model when config.model != ""
prompt prepend
`)
	userSlice := parse(t, `
CONFIG model = "opus"
REPO "https://github.com/org/project"
BRANCH "main"
`)
	merged := merge.Merge(base, userSlice)
	src := Serialize(merged)

	reparsed := parse(t, src)
	ctx := eval.NewContext(map[string]interface{}{
		"config": map[string]interface{}{
			"model":       "opus",
			"mcp_enabled": true,
		},
		"mcp": map[string]interface{}{
			"enabled": true,
		},
	})
	require.NoError(t, eval.Eval(reparsed, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Equal(t, "claude", ctx.Result.LaunchCommand)
	assert.Contains(t, ctx.Result.LaunchArgs, "--model")
	assert.Contains(t, ctx.Result.LaunchArgs, "opus")
	assert.Equal(t, "prepend", ctx.Result.PromptPosition)
	assert.Equal(t, "https://github.com/org/project", ctx.Result.Sandbox.RepoURL)
	assert.Equal(t, "main", ctx.Result.Sandbox.Branch)
}

func TestMergeSerialize_RemoveStmt(t *testing.T) {
	base := parse(t, `
AGENT test
arg "--verbose"
arg "--model" "opus"
`)
	slice := parse(t, `remove arg "--verbose"`)
	merged := merge.Merge(base, slice)
	src := Serialize(merged)

	reparsed := parse(t, src)
	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(reparsed, ctx))
	eval.ApplyRemoves(ctx.Result)
	assert.Equal(t, []string{"--model", "opus"}, ctx.Result.LaunchArgs)
}

func TestMergeSerialize_Recursive(t *testing.T) {
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
	merged := merge.Merge(merge.Merge(base, layer1), layer2)
	src := Serialize(merged)

	reparsed := parse(t, src)
	spec := extract.Extract(reparsed)

	assert.Equal(t, "claude", spec.Agent.Command)
	assert.Equal(t, "opus", spec.Config[0].Default)
	assert.Equal(t, []string{"am-delegate", "am-channel"}, spec.Skills)
	require.NotNil(t, spec.Repo)
	assert.Len(t, reparsed.Statements, 3)
}

func TestMergeSerialize_SetupPreserved(t *testing.T) {
	base := parse(t, "AGENT test\nSETUP timeout=60 <<SCRIPT\napt update\nSCRIPT\n")
	slice := parse(t, `CONFIG model = "opus"`)
	merged := merge.Merge(base, slice)
	src := Serialize(merged)

	reparsed := parse(t, src)
	spec := extract.Extract(reparsed)
	require.NotNil(t, spec.Setup)
	assert.Equal(t, 60, spec.Setup.Timeout)
	assert.Contains(t, spec.Setup.Script, "apt update")
}

func TestSerialize_EmptyProgram(t *testing.T) {
	prog := &parser.Program{}
	assert.Equal(t, "", Serialize(prog))
}
