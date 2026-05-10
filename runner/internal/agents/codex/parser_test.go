package codex

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var epoch = time.Time{}

func TestCodexParser_ParseNestedFormat(t *testing.T) {
	dir := t.TempDir()

	sessionDir := filepath.Join(dir, "sessions", "sess1")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	jsonl := `{"type":"assistant","message":{"model":"gpt-4o","usage":{"input_tokens":500,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":100}}}
{"type":"user","message":{"content":"test"}}
`
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "session.jsonl"), []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile(filepath.Join(sessionDir, "session.jsonl"), usage)
	require.NoError(t, err)

	assert.Len(t, usage.Models, 1)
	m := usage.Models["gpt-4o"]
	require.NotNil(t, m)
	assert.Equal(t, int64(500), m.InputTokens)
	assert.Equal(t, int64(200), m.OutputTokens)
	assert.Equal(t, int64(100), m.CacheReadTokens)
}

func TestCodexParser_ParseFlatFormat(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"model":"o3-mini","usage":{"input_tokens":1000,"output_tokens":400}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile(file, usage)
	require.NoError(t, err)

	assert.Len(t, usage.Models, 1)
	m := usage.Models["o3-mini"]
	require.NotNil(t, m)
	assert.Equal(t, int64(1000), m.InputTokens)
	assert.Equal(t, int64(400), m.OutputTokens)
}

func TestCodexSessionDirs_Priority(t *testing.T) {
	sandboxRoot := filepath.Join("sandbox", "root")
	dirs := codexSessionDirs(sandboxRoot)
	require.GreaterOrEqual(t, len(dirs), 1)
	assert.Equal(t, filepath.Join(sandboxRoot, "codex-home", "sessions"), dirs[0])
	if len(dirs) >= 2 {
		assert.Contains(t, dirs[1], filepath.Join(".codex", "sessions"))
	}
}

func TestCodexSessionDirs_EmptySandbox(t *testing.T) {
	dirs := codexSessionDirs("")
	for _, d := range dirs {
		assert.NotContains(t, d, "codex-home")
	}
}

func TestCodexParser_Parse_SandboxPath(t *testing.T) {
	sandboxRoot := t.TempDir()
	sessionDir := filepath.Join(sandboxRoot, "codex-home", "sessions", "2026", "03", "24")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	jsonl := `{"type":"assistant","message":{"model":"gpt-4.1","usage":{"input_tokens":100,"output_tokens":50}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "rollout-abc.jsonl"), []byte(jsonl), 0o644))

	parser := &codexParser{}
	usage, err := parser.Parse(sandboxRoot, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage, "should find sessions in sandbox codex-home")
	assert.Equal(t, int64(100), usage.Models["gpt-4.1"].InputTokens)
}

func TestCodexParser_ParseOpenAIResponseFormat(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"type":"response","response":{"model":"o3-mini","usage":{"prompt_tokens":800,"completion_tokens":350}}}
{"type":"response","response":{"model":"gpt-4.1","usage":{"prompt_tokens":1200,"completion_tokens":600}}}
{"type":"response","response":{"model":"o3-mini","usage":{"prompt_tokens":200,"completion_tokens":100}}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	assert.Len(t, usage.Models, 2)

	o3 := usage.Models["o3-mini"]
	require.NotNil(t, o3)
	assert.Equal(t, int64(1000), o3.InputTokens)
	assert.Equal(t, int64(450), o3.OutputTokens)

	gpt := usage.Models["gpt-4.1"]
	require.NotNil(t, gpt)
	assert.Equal(t, int64(1200), gpt.InputTokens)
	assert.Equal(t, int64(600), gpt.OutputTokens)
}

func TestCodexParser_ParseFlatOpenAIFormat(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"model":"o3-mini","usage":{"prompt_tokens":500,"completion_tokens":250}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	m := usage.Models["o3-mini"]
	require.NotNil(t, m)
	assert.Equal(t, int64(500), m.InputTokens)
	assert.Equal(t, int64(250), m.OutputTokens)
}

// Empty model in any usage-bearing branch must fall back to "codex-unknown"
// rather than silently drop tokens. Silent drop on malformed entries is
// what made #146 ship unfixed for weeks.
func TestCodexParser_EmptyModelFallsBackToUnknown(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"type":"response","response":{"usage":{"prompt_tokens":300,"completion_tokens":150}}}
{"usage":{"prompt_tokens":100,"completion_tokens":50}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	m := usage.Models["codex-unknown"]
	require.NotNil(t, m, "empty model branches must aggregate under codex-unknown")
	assert.Equal(t, int64(400), m.InputTokens)
	assert.Equal(t, int64(200), m.OutputTokens)
}

// When a single line has an empty-model message branch with positive tokens
// AND a populated-model response branch, the message branch wins (current
// precedence) and tokens land on codex-unknown — not on the response.model.
// This is documented by-design behavior; the test pins it so a future
// refactor that flips precedence (or skips empty-model branches) fails
// loudly instead of silently changing attribution.
func TestCodexParser_HybridLine_EmptyMessageModelWinsAsUnknown(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"type":"hybrid","message":{"usage":{"input_tokens":50,"output_tokens":20}},"response":{"model":"o3-mini","usage":{"prompt_tokens":999,"completion_tokens":888}}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	assert.Len(t, usage.Models, 1, "exactly one branch must be selected per line")
	m := usage.Models["codex-unknown"]
	require.NotNil(t, m, "message branch (empty model) wins precedence and falls back to codex-unknown")
	assert.Equal(t, int64(50), m.InputTokens)
	assert.Equal(t, int64(20), m.OutputTokens)
	assert.Nil(t, usage.Models["o3-mini"], "response branch must not contribute when message already matched")
}

// Anthropic-style fields must win when both naming styles coexist on a
// single entry — guards the precedence rule in effectiveInput/effectiveOutput.
func TestCodexParser_AnthropicFieldsWinOverOpenAI(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"type":"response","response":{"model":"o3-mini","usage":{"input_tokens":111,"output_tokens":22,"prompt_tokens":999,"completion_tokens":888}}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	m := usage.Models["o3-mini"]
	require.NotNil(t, m)
	assert.Equal(t, int64(111), m.InputTokens)
	assert.Equal(t, int64(22), m.OutputTokens)
}

// If a single line carries both a message and a response wrapper with
// positive counts, only the message branch contributes — guards against
// silent double-counting if Codex CLI ever emits hybrid lines.
func TestCodexParser_MessageWinsOverResponseOnSameLine(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "session.jsonl")

	jsonl := `{"type":"hybrid","message":{"model":"gpt-4o","usage":{"input_tokens":100,"output_tokens":50}},"response":{"model":"o3-mini","usage":{"prompt_tokens":999,"completion_tokens":888}}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	require.NoError(t, parseCodexJSONLFile(file, usage))

	assert.Len(t, usage.Models, 1, "must not aggregate across both branches")
	gpt := usage.Models["gpt-4o"]
	require.NotNil(t, gpt)
	assert.Equal(t, int64(100), gpt.InputTokens)
	assert.Equal(t, int64(50), gpt.OutputTokens)
	assert.Nil(t, usage.Models["o3-mini"], "response branch must not contribute when message already matched")
}
