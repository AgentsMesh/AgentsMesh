package claude

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

var epoch = time.Time{}

func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

func setupClaudeHomeProject(t *testing.T, sandboxPath string) string {
	t.Helper()
	home := t.TempDir()
	setHome(t, home)

	resolved, err := filepath.EvalSymlinks(sandboxPath)
	require.NoError(t, err)

	hash := claudePathHash(resolved)
	projectDir := filepath.Join(home, ".claude", "projects", hash)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	return projectDir
}

func TestClaudeParser_ParseJSONLFile(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, ".claude", "projects", "testproject")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	jsonl := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":100,"cache_read_input_tokens":200}}}
{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":2000,"output_tokens":800,"cache_creation_input_tokens":0,"cache_read_input_tokens":300}}}
{"type":"assistant","message":{"model":"claude-opus-4-20250514","usage":{"input_tokens":500,"output_tokens":200,"cache_creation_input_tokens":50,"cache_read_input_tokens":0}}}
invalid json line
{"type":"assistant","message":{"model":"","usage":{"input_tokens":10,"output_tokens":5}}}
`
	filePath := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(jsonl), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(filePath, usage)
	require.NoError(t, err)

	assert.Len(t, usage.Models, 2)

	sonnet := usage.Models["claude-sonnet-4-20250514"]
	require.NotNil(t, sonnet)
	assert.Equal(t, int64(3000), sonnet.InputTokens)
	assert.Equal(t, int64(1300), sonnet.OutputTokens)
	assert.Equal(t, int64(100), sonnet.CacheCreationTokens)
	assert.Equal(t, int64(500), sonnet.CacheReadTokens)

	opus := usage.Models["claude-opus-4-20250514"]
	require.NotNil(t, opus)
	assert.Equal(t, int64(500), opus.InputTokens)
	assert.Equal(t, int64(200), opus.OutputTokens)
}

func TestClaudeParser_Parse_HomeBasedPath(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 1)
	assert.Equal(t, int64(100), usage.Models["claude-sonnet-4-20250514"].InputTokens)
}

func TestClaudeParser_Parse_WorkspaceSubdir(t *testing.T) {
	sandbox := t.TempDir()
	wsDir := filepath.Join(sandbox, "workspace")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	projectDir := setupClaudeHomeProject(t, wsDir)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":200,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 1)
	assert.Equal(t, int64(200), usage.Models["claude-sonnet-4-20250514"].InputTokens)
}

func TestClaudeParser_Parse_SubagentFiles(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	subagentDir := filepath.Join(projectDir, "subagents", "abc123")
	require.NoError(t, os.MkdirAll(subagentDir, 0o755))
	subagentJSONL := `{"type":"assistant","message":{"model":"claude-haiku-3-20250514","usage":{"input_tokens":300,"output_tokens":150,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(subagentDir, "subagent.jsonl"), []byte(subagentJSONL), 0o644))

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 2)
	assert.Equal(t, int64(100), usage.Models["claude-sonnet-4-20250514"].InputTokens)
	assert.Equal(t, int64(300), usage.Models["claude-haiku-3-20250514"].InputTokens)
}

func TestClaudeParser_NoFiles(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir)
	parser := &claudeParser{}
	usage, err := parser.Parse(t.TempDir(), epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestClaudeParser_SkipsOldFiles(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50}}}`
	filePath := filepath.Join(projectDir, "old-session.jsonl")
	require.NoError(t, os.WriteFile(filePath, []byte(jsonl), 0o644))

	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(filePath, pastTime, pastTime))

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, time.Now().Add(-1*time.Minute))
	require.NoError(t, err)
	assert.Nil(t, usage)
}

func TestClaudePathHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/tmp/sandbox", "-tmp-sandbox"},
		{"/home/user/workspace", "-home-user-workspace"},
		{"/", "-"},
	}
	for _, tt := range tests {
		got := claudePathHash(tt.input)
		assert.Equal(t, tt.want, got, "claudePathHash(%q)", tt.input)
	}

	assert.Equal(t, "C-Users-test", claudePathHash(`C:\Users\test`))
	assert.Equal(t, "D-workspace", claudePathHash(`D:\workspace`))
}
