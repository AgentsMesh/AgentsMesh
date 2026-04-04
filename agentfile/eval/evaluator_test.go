package eval

import (
	"testing"

	"github.com/anthropics/agentsmesh/agentfile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEval_ArgStatements(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT claude

arg "--model" config.model when config.model != ""
arg "--verbose"
arg "--skip" when config.skip
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{
			"model": "sonnet",
			"skip":  false,
		},
	})
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "claude", ctx.Result.LaunchCommand)
	assert.Equal(t, []string{"--model", "sonnet", "--verbose"}, ctx.Result.LaunchArgs)
}

func TestEval_IfElse(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT claude

if config.mode == "plan" {
  arg "--permission-mode" "plan"
} else {
  arg "--default-mode"
}
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"mode": "plan"},
	})
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"--permission-mode", "plan"}, ctx.Result.LaunchArgs)

	// Test else branch
	ctx2 := NewContext(map[string]interface{}{
		"config": map[string]interface{}{"mode": "default"},
	})
	require.NoError(t, Eval(prog, ctx2))
	assert.Equal(t, []string{"--default-mode"}, ctx2.Result.LaunchArgs)
}

func TestEval_FileAndMkdir(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
plugin_dir = sandbox.root + "/plugin"
mkdir plugin_dir
file plugin_dir + "/config.json" json({ name: "test", version: "1.0" })
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"sandbox": map[string]interface{}{"root": "/tmp/sandbox"},
	})
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, []string{"/tmp/sandbox/plugin"}, ctx.Result.Dirs)
	require.Len(t, ctx.Result.FilesToCreate, 1)
	assert.Equal(t, "/tmp/sandbox/plugin/config.json", ctx.Result.FilesToCreate[0].Path)
	assert.Contains(t, ctx.Result.FilesToCreate[0].Content, `"name":"test"`)
}

func TestEval_Prompt(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT gemini
PROMPT_POSITION append
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, "append", ctx.Result.PromptPosition)
}

func TestEval_EnvDecl(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV TERM = "xterm-256color"
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, "xterm-256color", ctx.Result.EnvVars["TERM"])
}

func TestEval_CredentialEnvInjection(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV ANTHROPIC_API_KEY SECRET
ENV ANTHROPIC_BASE_URL TEXT OPTIONAL
ENV TERM = "xterm-256color"
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	ctx.Credentials = map[string]string{
		"ANTHROPIC_API_KEY":  "sk-secret-123",
		"ANTHROPIC_BASE_URL": "https://custom.api.com",
	}
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, "sk-secret-123", ctx.Result.EnvVars["ANTHROPIC_API_KEY"])
	assert.Equal(t, "https://custom.api.com", ctx.Result.EnvVars["ANTHROPIC_BASE_URL"])
	assert.Equal(t, "xterm-256color", ctx.Result.EnvVars["TERM"])
}

func TestEval_CredentialEnvRunnerHost(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
ENV ANTHROPIC_API_KEY SECRET
ENV TERM = "xterm-256color"
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	ctx.Credentials = map[string]string{"ANTHROPIC_API_KEY": "sk-secret"}
	ctx.IsRunnerHost = true
	require.NoError(t, Eval(prog, ctx))

	// RunnerHost mode: credentials NOT injected
	_, hasKey := ctx.Result.EnvVars["ANTHROPIC_API_KEY"]
	assert.False(t, hasKey)
	// Fixed value env still works
	assert.Equal(t, "xterm-256color", ctx.Result.EnvVars["TERM"])
}

func TestEval_Assignment(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
x = "hello"
y = x + " world"
arg y
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"hello world"}, ctx.Result.LaunchArgs)
}

func TestEval_ForLoop(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
for name, server in servers {
  arg "-c" name + "=" + server.url
}
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"servers": map[string]interface{}{
			"agentsmesh": map[string]interface{}{"url": "http://localhost:19000/mcp"},
		},
	})
	require.NoError(t, Eval(prog, ctx))

	require.Len(t, ctx.Result.LaunchArgs, 2)
	assert.Equal(t, "-c", ctx.Result.LaunchArgs[0])
	assert.Equal(t, "agentsmesh=http://localhost:19000/mcp", ctx.Result.LaunchArgs[1])
}

func TestEval_StringInterpolation(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
file "/out.json" <<EOF
{
  "url": "http://127.0.0.1:${mcp.port}/mcp",
  "headers": {"X-Pod-Key": "${pod.key}"}
}
EOF
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"mcp": map[string]interface{}{"port": float64(19000)},
		"pod": map[string]interface{}{"key": "pod-abc-123"},
	})
	require.NoError(t, Eval(prog, ctx))

	require.Len(t, ctx.Result.FilesToCreate, 1)
	content := ctx.Result.FilesToCreate[0].Content
	assert.Contains(t, content, "http://127.0.0.1:19000/mcp")
	assert.Contains(t, content, "pod-abc-123")
	assert.NotContains(t, content, "${mcp.port}")
	assert.NotContains(t, content, "${pod.key}")
}

func TestEval_StringInterpolationInStringLit(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
arg "--url" "http://${host}:${port}/api"
`)
	require.Empty(t, errs)

	ctx := NewContext(map[string]interface{}{
		"host": "localhost",
		"port": "8080",
	})
	require.NoError(t, Eval(prog, ctx))

	assert.Equal(t, []string{"--url", "http://localhost:8080/api"}, ctx.Result.LaunchArgs)
}

func TestEval_ListLiteral(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
items = ["a", "b", "c"]
for item in items {
  arg item
}
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"a", "b", "c"}, ctx.Result.LaunchArgs)
}

func TestEval_ForListWithIndex(t *testing.T) {
	prog, errs := parser.Parse(`
AGENT test
items = ["x", "y"]
for i, item in items {
  arg item
}
`)
	require.Empty(t, errs)

	ctx := NewContext(nil)
	require.NoError(t, Eval(prog, ctx))
	assert.Equal(t, []string{"x", "y"}, ctx.Result.LaunchArgs)
}
