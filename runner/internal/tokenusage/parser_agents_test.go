package tokenusage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Aider parser tests ---

func TestAiderParser_Parse(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	history := `# Chat History

## User
Hello

## Assistant
Hi there!

> Tokens: 12k sent, 3.4k received, 45k cache write, 123k cache read

## User
Do something

## Assistant
Done.

> Tokens: 1,234 sent, 567 received

Some other line
> Not a token line
`
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, ".aider.chat.history.md"), []byte(history), 0o644))

	parser := &AiderParser{}
	usage, err := parser.Parse(dir, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)

	m := usage.Models["aider-unknown"]
	require.NotNil(t, m)
	// 12000 + 1234 = 13234
	assert.Equal(t, int64(13234), m.InputTokens)
	// 3400 + 567 = 3967
	assert.Equal(t, int64(3967), m.OutputTokens)
	assert.Equal(t, int64(45000), m.CacheCreationTokens)
	assert.Equal(t, int64(123000), m.CacheReadTokens)
}

func TestAiderParser_SkipsOldFiles(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	history := "> Tokens: 500 sent, 200 received\n"
	filePath := filepath.Join(wsDir, ".aider.chat.history.md")
	require.NoError(t, os.WriteFile(filePath, []byte(history), 0o644))

	// Set file mtime to the past
	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(filePath, pastTime, pastTime))

	parser := &AiderParser{}
	// podStartedAt is recent, so old file should be skipped
	usage, err := parser.Parse(dir, time.Now().Add(-1*time.Minute))
	require.NoError(t, err)
	assert.Nil(t, usage, "old aider history file should be skipped")
}

func TestAiderParser_NoFile(t *testing.T) {
	dir := t.TempDir()
	parser := &AiderParser{}
	usage, err := parser.Parse(dir, epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestParseTokenValue(t *testing.T) {
	tests := []struct {
		num    string
		suffix string
		want   int64
	}{
		{"12", "", 12},
		{"12", "k", 12000},
		{"3.4", "k", 3400},
		{"1,234", "", 1234},
		{"1.5", "m", 1500000},
		{"0", "", 0},
		{"abc", "", 0},
	}

	for _, tt := range tests {
		got := parseTokenValue(tt.num, tt.suffix)
		assert.Equal(t, tt.want, got, "parseTokenValue(%q, %q)", tt.num, tt.suffix)
	}
}

// --- OpenCode parser tests ---

func TestOpenCodeParser_ParsePrimaryUsage(t *testing.T) {
	dir := t.TempDir()
	msgDir := filepath.Join(dir, "sess1")
	require.NoError(t, os.MkdirAll(msgDir, 0o755))

	msg := `{"model":"claude-3-haiku","usage":{"input_tokens":800,"output_tokens":300,"cache_creation_input_tokens":50,"cache_read_input_tokens":100}}`
	require.NoError(t, os.WriteFile(filepath.Join(msgDir, "msg_001.json"), []byte(msg), 0o644))

	usage := NewTokenUsage()
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

	usage := NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	require.NoError(t, err)

	m := usage.Models["gpt-4"]
	require.NotNil(t, m)
	assert.Equal(t, int64(1000), m.InputTokens)
	assert.Equal(t, int64(500), m.OutputTokens)
	assert.Equal(t, int64(200), m.CacheReadTokens)
}

// --- Registry tests ---

func TestGetParser(t *testing.T) {
	tests := []struct {
		agent   string
		wantNil bool
	}{
		{"claude", false},
		{"claude-code", false},
		{"codex", false},
		{"codex-cli", false},
		{"aider", false},
		{"opencode", false},
		{"unknown-agent", true},
		{"/usr/bin/claude", false},
		{"C:\\bin\\claude", false},
		{"Claude", false}, // case insensitive
		{"AIDER", false},
	}

	for _, tt := range tests {
		parser := GetParser(tt.agent)
		if tt.wantNil {
			assert.Nil(t, parser, "GetParser(%q) should be nil", tt.agent)
		} else {
			assert.NotNil(t, parser, "GetParser(%q) should not be nil", tt.agent)
		}
	}
}

// --- Collector tests ---

func TestCollect_UnknownAgent(t *testing.T) {
	usage := Collect("unknown-agent", t.TempDir(), epoch)
	assert.Nil(t, usage)
}

func TestCollect_NoData(t *testing.T) {
	setHome(t, t.TempDir())
	usage := Collect("claude", t.TempDir(), epoch)
	assert.Nil(t, usage)
}

func TestCollect_WithData(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "s.jsonl"), []byte(jsonl), 0o644))

	usage := Collect("claude", sandbox, epoch)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 1)
}

// --- mtime filter tests ---

func TestIsModifiedAfter(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

	// File just created, should be after epoch
	assert.True(t, isModifiedAfter(file, epoch))

	// File should be after a time far in the past
	assert.True(t, isModifiedAfter(file, time.Now().Add(-24*time.Hour)))

	// File should not be after a future time
	assert.False(t, isModifiedAfter(file, time.Now().Add(24*time.Hour)))

	// Non-existent file
	assert.False(t, isModifiedAfter(filepath.Join(dir, "nonexistent"), epoch))
}
