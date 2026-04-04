package merge

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// P1: Empty merge
func TestMerge_EmptyBaseAndSlice(t *testing.T) {
	base := parse(t, "")
	slice := parse(t, "")
	Merge(base, slice)
	assert.Empty(t, base.Declarations)
	assert.Empty(t, base.Statements)
}

// P1: Slice adds new declarations not in base
func TestMerge_SliceAddsNew(t *testing.T) {
	base := parse(t, `AGENT test`)
	slice := parse(t, `
REPO "https://github.com/org/repo"
SETUP timeout=60 <<EOF
npm install
EOF
`)
	Merge(base, slice)
	spec := extract.Extract(base)

	assert.Equal(t, "test", spec.Agent.Command)
	require.NotNil(t, spec.Repo)
	assert.Equal(t, "https://github.com/org/repo", spec.Repo.URL)
	require.NotNil(t, spec.Setup)
	assert.Equal(t, 60, spec.Setup.Timeout)
}

// P1: EXECUTABLE override
func TestMerge_ExecutableOverride(t *testing.T) {
	base := parse(t, `
AGENT test
EXECUTABLE test-bin
`)
	slice := parse(t, `EXECUTABLE new-bin`)
	Merge(base, slice)
	spec := extract.Extract(base)
	assert.Equal(t, "new-bin", spec.Agent.Executable)
}

// P1: SETUP override
func TestMerge_SetupOverride(t *testing.T) {
	base := parse(t, `
AGENT test
SETUP timeout=300 <<EOF
npm install
EOF
`)
	slice := parse(t, `
SETUP timeout=60 <<EOF
yarn install
EOF
`)
	Merge(base, slice)
	spec := extract.Extract(base)
	assert.Equal(t, 60, spec.Setup.Timeout)
	assert.Contains(t, spec.Setup.Script, "yarn")
}

// P1: REMOVE CONFIG in merge
func TestMerge_RemoveConfig(t *testing.T) {
	base := parse(t, `
AGENT test
CONFIG model SELECT("sonnet", "opus") = "sonnet"
CONFIG debug BOOL = false
`)
	slice := parse(t, `REMOVE CONFIG debug`)
	Merge(base, slice)
	spec := extract.Extract(base)

	// debug removed, only model remains
	require.Len(t, spec.Config, 1)
	assert.Equal(t, "model", spec.Config[0].Name)
}

// P1: Complete multi-layer merge → eval → ApplyRemoves E2E
func TestMerge_E2E_ThreeLayer(t *testing.T) {
	// Layer 0: Agent base
	base := parse(t, `
AGENT claude
EXECUTABLE claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG mcp_enabled BOOL = true
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
MCP ON
SKILLS am-delegate, am-channel

PROMPT_POSITION prepend

arg "--model" config.model when config.model != ""

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  plugin_dir = sandbox.root + "/agentsmesh-plugin"
  mkdir plugin_dir
  file plugin_dir + "/.mcp.json" json({ mcpServers: mcp_cfg })
  arg "--plugin-dir" plugin_dir
}
`)

	// Layer 1: Org defaults
	orgSlice := parse(t, `
CONFIG model = "sonnet"
REMOVE ENV ANTHROPIC_BASE_URL
SKILLS org-custom-skill
`)

	// Layer 2: User instance
	userSlice := parse(t, `
CONFIG model = "opus"
REPO "https://github.com/org/project"
BRANCH "main"
`)

	// Recursive merge
	Merge(base, orgSlice)
	Merge(base, userSlice)

	// Verify declarations
	spec := extract.Extract(base)
	assert.Equal(t, "claude", spec.Agent.Command)
	assert.Equal(t, "opus", spec.Config[0].Default) // user overrode org
	assert.Len(t, spec.Env, 1) // BASE_URL removed by org
	assert.Equal(t, "ANTHROPIC_API_KEY", spec.Env[0].Name)
	assert.Equal(t, []string{"am-delegate", "am-channel", "org-custom-skill"}, spec.Skills)

	// Eval with merged config
	ctx := eval.NewContext(map[string]interface{}{
		"config": map[string]interface{}{
			"model":       "opus",
			"mcp_enabled": true,
		},
		"mcp": map[string]interface{}{
			"enabled":   true,
			"builtin":   map[string]interface{}{"agentsmesh": map[string]interface{}{"url": "http://localhost:19000"}},
			"installed": map[string]interface{}{},
		},
		"sandbox": map[string]interface{}{"root": "/sandbox"},
	})
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	r := ctx.Result
	assert.Equal(t, "claude", r.LaunchCommand)
	assert.Contains(t, r.LaunchArgs, "--model")
	assert.Contains(t, r.LaunchArgs, "opus")
	assert.Contains(t, r.LaunchArgs, "--plugin-dir")
	assert.Equal(t, "https://github.com/org/project", r.Sandbox.RepoURL)
	assert.Equal(t, "main", r.Sandbox.Branch)
}

// P2: REMOVE non-existent item (should not error)
func TestMerge_RemoveNonExistent(t *testing.T) {
	base := parse(t, `
AGENT test
ENV KEY1 SECRET
`)
	slice := parse(t, `REMOVE ENV NONEXISTENT`)
	Merge(base, slice)

	ctx := eval.NewContext(nil)
	ctx.Credentials = map[string]string{"KEY1": "val"}
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)
	// Should not error, KEY1 still present
	assert.Equal(t, "val", ctx.Result.EnvVars["KEY1"])
}

// P2: Slice with only statements (no declarations)
func TestMerge_SliceOnlyStatements(t *testing.T) {
	base := parse(t, `
AGENT test
arg "--base"
`)
	slice := parse(t, `
arg "--extra"
REMOVE arg "--base"
`)
	Merge(base, slice)

	ctx := eval.NewContext(nil)
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Equal(t, []string{"--extra"}, ctx.Result.LaunchArgs)
}
