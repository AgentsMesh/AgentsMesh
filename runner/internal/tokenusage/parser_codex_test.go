package tokenusage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Codex parser tests ---

func TestCodexParser_ParseNestedFormat(t *testing.T) {
	dir := t.TempDir()

	// Create a fake HOME-based codex session dir
	sessionDir := filepath.Join(dir, "sessions", "sess1")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	jsonl := `{"type":"assistant","message":{"model":"gpt-4o","usage":{"input_tokens":500,"output_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":100}}}
{"type":"user","message":{"content":"test"}}
`
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "session.jsonl"), []byte(jsonl), 0o644))

	usage := NewTokenUsage()
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

	usage := NewTokenUsage()
	err := parseCodexJSONLFile(file, usage)
	require.NoError(t, err)

	assert.Len(t, usage.Models, 1)
	m := usage.Models["o3-mini"]
	require.NotNil(t, m)
	assert.Equal(t, int64(1000), m.InputTokens)
	assert.Equal(t, int64(400), m.OutputTokens)
}

func TestCodexSessionDirs_Priority(t *testing.T) {
	// Should return sandbox codex-home first, then user-level
	sandboxRoot := filepath.Join("sandbox", "root")
	dirs := codexSessionDirs(sandboxRoot)
	require.GreaterOrEqual(t, len(dirs), 1)
	assert.Equal(t, filepath.Join(sandboxRoot, "codex-home", "sessions"), dirs[0])
	// Second entry should be user-level
	if len(dirs) >= 2 {
		assert.Contains(t, dirs[1], filepath.Join(".codex", "sessions"))
	}
}

func TestCodexSessionDirs_EmptySandbox(t *testing.T) {
	dirs := codexSessionDirs("")
	// No sandbox path → only user-level
	for _, d := range dirs {
		assert.NotContains(t, d, "codex-home")
	}
}

func TestCodexParser_Parse_SandboxPath(t *testing.T) {
	// Simulate sandbox with codex-home/sessions/ containing a session file
	sandboxRoot := t.TempDir()
	sessionDir := filepath.Join(sandboxRoot, "codex-home", "sessions", "2026", "03", "24")
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	jsonl := `{"type":"assistant","message":{"model":"gpt-4.1","usage":{"input_tokens":100,"output_tokens":50}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "rollout-abc.jsonl"), []byte(jsonl), 0o644))

	parser := &CodexParser{}
	usage, err := parser.Parse(sandboxRoot, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage, "should find sessions in sandbox codex-home")
	assert.Equal(t, int64(100), usage.Models["gpt-4.1"].InputTokens)
}
