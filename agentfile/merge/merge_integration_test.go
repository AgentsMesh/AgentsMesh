package merge

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/eval"
	"github.com/anthropics/agentsmesh/agentfile/extract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Integration: Each Agent + Layer
// ============================================================

func evalMerged(t *testing.T, base, slice string, vars map[string]interface{}) *eval.BuildResult {
	t.Helper()
	baseProg := parse(t, base)
	sliceProg := parse(t, slice)
	Merge(baseProg, sliceProg)
	ctx := eval.NewContext(vars)
	require.NoError(t, eval.Eval(baseProg, ctx))
	eval.ApplyRemoves(ctx.Result)
	return ctx.Result
}

func mcpVars() map[string]interface{} {
	return map[string]interface{}{
		"mcp": map[string]interface{}{
			"enabled":   true,
			"builtin":   map[string]interface{}{"agentsmesh": map[string]interface{}{"type": "http", "url": "http://127.0.0.1:19000/mcp"}},
			"installed": map[string]interface{}{},
		},
		"sandbox": map[string]interface{}{"root": "/sandbox", "work_dir": "/sandbox/workspace"},
	}
}

const geminiBase = `
AGENT gemini
EXECUTABLE gemini
CONFIG mcp_enabled BOOL = true
CONFIG sandbox_mode BOOL = false
ENV GOOGLE_API_KEY SECRET OPTIONAL
MCP ON

arg "--sandbox" when config.sandbox_mode
PROMPT_POSITION append

if mcp.enabled {
  mcp_cfg = mcp_transform(json_merge(mcp.builtin, mcp.installed), "gemini")
  mkdir sandbox.work_dir + "/.gemini"
  file sandbox.work_dir + "/.gemini/settings.json" json({ mcpServers: mcp_cfg })
}
`

func TestIntegration_GeminiWithLayer(t *testing.T) {
	vars := mcpVars()
	vars["config"] = map[string]interface{}{"mcp_enabled": true, "sandbox_mode": true}

	r := evalMerged(t, geminiBase, `
CONFIG sandbox_mode = true
REPO "https://github.com/org/project"
BRANCH "develop"
`, vars)

	assert.Equal(t, "gemini", r.LaunchCommand)
	assert.Equal(t, "append", r.PromptPosition)
	assert.Contains(t, r.LaunchArgs, "--sandbox")
	assert.Equal(t, "https://github.com/org/project", r.Sandbox.RepoURL)
	assert.Equal(t, "develop", r.Sandbox.Branch)
	// Gemini MCP format: httpUrl
	for _, f := range r.FilesToCreate {
		if f.Content != "" {
			assert.Contains(t, f.Content, "httpUrl")
		}
	}
}

const codexBase = `
AGENT codex
EXECUTABLE codex
CONFIG mcp_enabled BOOL = true
CONFIG approval_mode SELECT("suggest", "auto-edit", "full-auto") = "suggest"
ENV OPENAI_API_KEY SECRET OPTIONAL
MCP ON

arg "--approval-mode" config.approval_mode when config.approval_mode != ""
PROMPT_POSITION prepend

if mcp.enabled {
  mcp_cfg = json_merge(mcp.builtin, mcp.installed)
  mkdir sandbox.work_dir + "/.codex"
  file sandbox.work_dir + "/.codex/mcp.json" json({ mcpServers: mcp_cfg })
}
`

func TestIntegration_CodexWithLayer(t *testing.T) {
	vars := mcpVars()
	vars["config"] = map[string]interface{}{"mcp_enabled": true, "approval_mode": "full-auto"}

	r := evalMerged(t, codexBase, `CONFIG approval_mode = "full-auto"`, vars)

	assert.Equal(t, "codex", r.LaunchCommand)
	assert.Contains(t, r.LaunchArgs, "--approval-mode")
	assert.Contains(t, r.LaunchArgs, "full-auto")
	assert.True(t, len(r.FilesToCreate) >= 1)
}

const aiderBase = `
AGENT aider
EXECUTABLE aider
CONFIG model STRING = ""
CONFIG edit_format SELECT("", "whole", "diff", "udiff") = ""
ENV OPENAI_API_KEY SECRET OPTIONAL
MCP OFF

arg "--model" config.model when config.model != ""
arg "--edit-format" config.edit_format when config.edit_format != ""
PROMPT_POSITION none
`

func TestIntegration_AiderWithLayer(t *testing.T) {
	vars := map[string]interface{}{
		"config": map[string]interface{}{"model": "gpt-4", "edit_format": "diff"},
	}
	r := evalMerged(t, aiderBase, `
CONFIG model = "gpt-4"
CONFIG edit_format = "diff"
REPO "https://github.com/org/repo"
`, vars)

	assert.Equal(t, "aider", r.LaunchCommand)
	assert.Equal(t, "none", r.PromptPosition)
	assert.False(t, r.MCPEnabled)
	assert.Contains(t, r.LaunchArgs, "--model")
	assert.Contains(t, r.LaunchArgs, "gpt-4")
	assert.Contains(t, r.LaunchArgs, "--edit-format")
	assert.Contains(t, r.LaunchArgs, "diff")
	assert.Equal(t, "https://github.com/org/repo", r.Sandbox.RepoURL)
	assert.Empty(t, r.FilesToCreate)
}

// ============================================================
// Integration: Edge scenarios
// ============================================================

// Empty layer preserves base unchanged
func TestIntegration_EmptyLayer(t *testing.T) {
	vars := mcpVars()
	vars["config"] = map[string]interface{}{"mcp_enabled": true}

	baseOnly := func() *eval.BuildResult {
		prog := parse(t, geminiBase)
		ctx := eval.NewContext(vars)
		require.NoError(t, eval.Eval(prog, ctx))
		return ctx.Result
	}
	withEmpty := func() *eval.BuildResult {
		return evalMerged(t, geminiBase, "", vars)
	}

	a := baseOnly()
	b := withEmpty()
	assert.Equal(t, a.LaunchCommand, b.LaunchCommand)
	assert.Equal(t, a.PromptPosition, b.PromptPosition)
	assert.Equal(t, a.MCPEnabled, b.MCPEnabled)
	assert.Equal(t, len(a.LaunchArgs), len(b.LaunchArgs))
}

// Remove arg then add replacement
func TestIntegration_RemoveAndReplace(t *testing.T) {
	base := `
AGENT test
arg "--old-flag" "old-value"
arg "--keep"
`
	slice := `
REMOVE arg "--old-flag"
arg "--new-flag" "new-value"
`
	r := evalMerged(t, base, slice, nil)

	assert.NotContains(t, r.LaunchArgs, "--old-flag")
	assert.NotContains(t, r.LaunchArgs, "old-value")
	assert.Contains(t, r.LaunchArgs, "--keep")
	assert.Contains(t, r.LaunchArgs, "--new-flag")
	assert.Contains(t, r.LaunchArgs, "new-value")
}

// Usage pattern: PR Review Bot
func TestIntegration_UsagePattern_PRReviewBot(t *testing.T) {
	claudeBase := `
AGENT claude
EXECUTABLE claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG mcp_enabled BOOL = true
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
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
`
	prReviewSlice := `
CONFIG model = "sonnet"
REPO "https://github.com/myorg/myproject"
BRANCH "main"
GIT_CREDENTIAL oauth

arg "--permission-mode" "plan"
`
	vars := mcpVars()
	vars["config"] = map[string]interface{}{"model": "sonnet", "mcp_enabled": true}

	r := evalMerged(t, claudeBase, prReviewSlice, vars)

	assert.Equal(t, "claude", r.LaunchCommand)
	assert.Contains(t, r.LaunchArgs, "--model")
	assert.Contains(t, r.LaunchArgs, "sonnet")
	assert.Contains(t, r.LaunchArgs, "--permission-mode")
	assert.Contains(t, r.LaunchArgs, "plan")
	assert.Contains(t, r.LaunchArgs, "--plugin-dir")
	assert.Equal(t, "https://github.com/myorg/myproject", r.Sandbox.RepoURL)
	assert.Equal(t, "main", r.Sandbox.Branch)
	assert.Equal(t, "oauth", r.Sandbox.CredentialType)
	assert.Equal(t, []string{"am-delegate", "am-channel"}, r.Skills)
}

// 4-layer recursive merge
func TestIntegration_FourLayerRecursive(t *testing.T) {
	base := parse(t, `
AGENT claude
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission SELECT("default", "plan") = "default"
ENV ANTHROPIC_API_KEY SECRET OPTIONAL
SKILLS am-delegate

PROMPT_POSITION prepend

arg "--model" config.model when config.model != ""
`)
	orgLayer := parse(t, `
CONFIG model = "sonnet"
CONFIG permission = "plan"
SKILLS org-skill
`)
	teamLayer := parse(t, `
REMOVE SKILLS am-delegate
SKILLS team-skill
`)
	userLayer := parse(t, `
CONFIG model = "opus"
REPO "https://github.com/user/repo"
BRANCH "feature-branch"
`)

	Merge(base, orgLayer)
	Merge(base, teamLayer)
	Merge(base, userLayer)
	spec := extract.Extract(base)

	// model: base="" → org="sonnet" → user="opus"
	assert.Equal(t, "opus", spec.Config[0].Default)
	// permission: base="default" → org="plan"
	assert.Equal(t, "plan", spec.Config[1].Default)
	// skills: base=[am-delegate] → org adds [org-skill] → team removes am-delegate, adds team-skill
	// After extract (no ApplyRemoves): am-delegate removed by REMOVE decl in merge
	assert.Contains(t, spec.Skills, "org-skill")
	assert.Contains(t, spec.Skills, "team-skill")
	// repo from user layer
	assert.Equal(t, "https://github.com/user/repo", spec.Repo.URL)
	assert.Equal(t, "feature-branch", spec.Repo.Branch)

	// Eval
	ctx := eval.NewContext(map[string]interface{}{
		"config": map[string]interface{}{"model": "opus", "permission": "plan"},
	})
	require.NoError(t, eval.Eval(base, ctx))
	eval.ApplyRemoves(ctx.Result)

	assert.Contains(t, ctx.Result.LaunchArgs, "--model")
	assert.Contains(t, ctx.Result.LaunchArgs, "opus")
	// REMOVE SKILLS am-delegate applied
	assert.NotContains(t, ctx.Result.Skills, "am-delegate")
	assert.Contains(t, ctx.Result.Skills, "org-skill")
	assert.Contains(t, ctx.Result.Skills, "team-skill")
}

// Aider MCP OFF — layer cannot turn MCP ON (declaration override)
func TestIntegration_AiderMCPOverride(t *testing.T) {
	r := evalMerged(t, aiderBase, `MCP ON`, map[string]interface{}{
		"config": map[string]interface{}{"model": "", "edit_format": ""},
		"mcp":    map[string]interface{}{"enabled": true},
	})

	// MCP declaration overridden to ON by layer
	assert.True(t, r.MCPEnabled)
	// But base has no MCP build logic, so no files created
	assert.Empty(t, r.FilesToCreate)
}
