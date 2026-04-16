package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func TestParse_EvalSymlinksError(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	parser := &claudeParser{}
	usage, err := parser.Parse("/nonexistent/path/that/does/not/exist", epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestParse_ProjectDirNotExist(t *testing.T) {
	sandbox := t.TempDir()
	home := t.TempDir()
	setHome(t, home)

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestParseClaudeJSONLFile_ScannerError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.jsonl")

	line := make([]byte, 11*1024*1024)
	for i := range line {
		line[i] = 'x'
	}
	require.NoError(t, os.WriteFile(path, line, 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(path, usage)
	assert.Error(t, err, "expected scanner error for oversized line")
}

func TestParseClaudeJSONLFile_EmptyModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty_model.jsonl")
	content := `{"type":"assistant","message":{"model":"","usage":{"input_tokens":100,"output_tokens":50}}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(path, usage)
	require.NoError(t, err)
	assert.True(t, usage.IsEmpty(), "entries with empty model should be skipped")
}

func TestParseClaudeJSONLFile_ZeroTokens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zero.jsonl")
	content := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":0,"output_tokens":0}}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(path, usage)
	require.NoError(t, err)
	assert.True(t, usage.IsEmpty(), "entries with zero tokens should be skipped")
}

func TestParseClaudeJSONLFile_OpenError(t *testing.T) {
	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile("/nonexistent/file.jsonl", usage)
	assert.Error(t, err)
}

func TestParse_NonJSONLFilesSkipped(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "notes.txt"),
		[]byte("not a jsonl file"),
		0o644,
	))

	parser := &claudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestParseClaudeJSONLFile_NonAssistantType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")
	content := `{"type":"user","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"system","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50}}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(path, usage)
	require.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}
