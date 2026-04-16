package codex

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func TestSendControlRequest_ReturnsNotSupported(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, slog.Default())
	_, err := tr.SendControlRequest("", "", nil)
	assert.ErrorIs(t, err, acp.ErrControlNotSupported)
}

func TestClose_NoOp(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, slog.Default())
	tr.Close()
}

func TestRespondToPermission_InvalidID(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, slog.Default())
	err := tr.RespondToPermission("not-a-number", true, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse request ID")
}

func TestRegisterInit_TransportFactory(t *testing.T) {
	tr := acp.NewTransport(TransportType, acp.EventCallbacks{}, slog.Default())
	assert.NotNil(t, tr)
}

func TestMergeTomlMcpServers_ParentDirNotExist(t *testing.T) {
	err := mergeTomlMcpServers("/nonexistent/dir/config.toml", "[mcp_servers.s]\nurl = \"u\"\n")
	assert.Error(t, err)
}

func TestMergeTomlMcpServers_InvalidExistingToml(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(configPath, []byte("invalid toml [[["), 0644))

	err := mergeTomlMcpServers(configPath, "[mcp_servers.s]\nurl = \"u\"\n")
	assert.Error(t, err)
}

func TestMergeTomlMcpServers_InvalidPlatformToml(t *testing.T) {
	err := mergeTomlMcpServers("/tmp/nonexistent", "invalid [[[")
	assert.Error(t, err)
}

func TestCodexParser_Parse_NoSandbox(t *testing.T) {
	parser := &codexParser{}
	usage, err := parser.Parse("", time.Time{})
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestCodexParser_Parse_WithData(t *testing.T) {
	sandbox := t.TempDir()
	sessDir := filepath.Join(sandbox, "codex-home", "sessions", "2024", "01", "01")
	require.NoError(t, os.MkdirAll(sessDir, 0755))

	jsonl := `{"type":"assistant","message":{"model":"gpt-4","usage":{"input_tokens":100,"output_tokens":50}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(sessDir, "rollout-1.jsonl"), []byte(jsonl), 0644))

	parser := &codexParser{}
	usage, err := parser.Parse(sandbox, time.Time{})
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, int64(100), usage.Models["gpt-4"].InputTokens)
}

func TestCodexParser_ParseJSONLFile_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.jsonl")
	require.NoError(t, os.WriteFile(file, []byte("not json\n{}\n"), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestCodexParser_ParseJSONLFile_FlatStructure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "flat.jsonl")
	jsonl := `{"model":"gpt-4","usage":{"input_tokens":200,"output_tokens":100}}
`
	require.NoError(t, os.WriteFile(file, []byte(jsonl), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseCodexJSONLFile(file, usage)
	require.NoError(t, err)
	assert.Equal(t, int64(200), usage.Models["gpt-4"].InputTokens)
}

func TestCodexParser_SessionDirs_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", "")
	}
	dirs := codexSessionDirs("/sandbox")
	assert.Len(t, dirs, 1)
}

func TestCodexParser_ParseSessionsDir_SkipsNonJSONL(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0644))

	usage := tokenusage.NewTokenUsage()
	parseCodexSessionsDir(dir, time.Time{}, usage)
	assert.True(t, usage.IsEmpty())
}

func TestCodexParser_ParseSessionsDir_SkipsOldFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "old.jsonl")
	require.NoError(t, os.WriteFile(file, []byte(`{"model":"m","usage":{"input_tokens":1,"output_tokens":1}}`+"\n"), 0644))

	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(file, pastTime, pastTime))

	usage := tokenusage.NewTokenUsage()
	parseCodexSessionsDir(dir, time.Now().Add(-1*time.Minute), usage)
	assert.True(t, usage.IsEmpty())
}
