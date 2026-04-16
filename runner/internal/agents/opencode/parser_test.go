package opencode

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

func TestOpenCodeParser_Parse_WithData(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	msgDir := filepath.Join(home, ".local", "share", "opencode", "storage", "message", "sess1")
	require.NoError(t, os.MkdirAll(msgDir, 0o755))

	msg := `{"model":"claude-3-haiku","usage":{"input_tokens":800,"output_tokens":300}}`
	require.NoError(t, os.WriteFile(filepath.Join(msgDir, "msg_001.json"), []byte(msg), 0o644))

	parser := &opencodeParser{}
	usage, err := parser.Parse(t.TempDir(), epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, int64(800), usage.Models["claude-3-haiku"].InputTokens)
}

func TestOpenCodeParser_Parse_NoFiles(t *testing.T) {
	setHome(t, t.TempDir())
	parser := &opencodeParser{}
	usage, err := parser.Parse(t.TempDir(), epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestOpenCodeParser_Parse_SkipsOldFiles(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	msgDir := filepath.Join(home, ".local", "share", "opencode", "storage", "message", "sess1")
	require.NoError(t, os.MkdirAll(msgDir, 0o755))

	filePath := filepath.Join(msgDir, "msg_001.json")
	require.NoError(t, os.WriteFile(filePath, []byte(`{"model":"m","usage":{"input_tokens":1,"output_tokens":1}}`), 0o644))

	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(filePath, pastTime, pastTime))

	parser := &opencodeParser{}
	usage, err := parser.Parse(t.TempDir(), time.Now().Add(-1*time.Minute))
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestOpenCodeParser_ParsePrimaryUsage(t *testing.T) {
	dir := t.TempDir()
	msgDir := filepath.Join(dir, "sess1")
	require.NoError(t, os.MkdirAll(msgDir, 0o755))

	msg := `{"model":"claude-3-haiku","usage":{"input_tokens":800,"output_tokens":300,"cache_creation_input_tokens":50,"cache_read_input_tokens":100}}`
	require.NoError(t, os.WriteFile(filepath.Join(msgDir, "msg_001.json"), []byte(msg), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(filepath.Join(msgDir, "msg_001.json"), usage)
	require.NoError(t, err)

	m := usage.Models["claude-3-haiku"]
	require.NotNil(t, m)
	assert.Equal(t, int64(800), m.InputTokens)
	assert.Equal(t, int64(300), m.OutputTokens)
	assert.Equal(t, int64(50), m.CacheCreationTokens)
	assert.Equal(t, int64(100), m.CacheReadTokens)
}

func TestOpenCodeParser_ParseAlternativeUsage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "msg_002.json")

	msg := `{"model":"gpt-4","token_usage":{"prompt_tokens":1000,"completion_tokens":500,"cached_tokens":200}}`
	require.NoError(t, os.WriteFile(file, []byte(msg), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	require.NoError(t, err)

	m := usage.Models["gpt-4"]
	require.NotNil(t, m)
	assert.Equal(t, int64(1000), m.InputTokens)
	assert.Equal(t, int64(500), m.OutputTokens)
	assert.Equal(t, int64(200), m.CacheReadTokens)
}

func TestOpenCodeParser_ParseMalformedFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "msg_bad.json")
	require.NoError(t, os.WriteFile(file, []byte("not json"), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestOpenCodeParser_ParseNoModel(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "msg_nomodel.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"usage":{"input_tokens":10,"output_tokens":5}}`), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	assert.NoError(t, err)
	assert.NotNil(t, usage.Models["opencode-unknown"])
}
