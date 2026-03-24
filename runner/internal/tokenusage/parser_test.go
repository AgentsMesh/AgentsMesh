package tokenusage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// epoch is a zero time used as podStartedAt when we want all files to pass the mtime filter.
var epoch = time.Time{}

// setHome overrides the home directory for os.UserHomeDir() on all platforms.
// On Unix it sets HOME; on Windows it sets USERPROFILE.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

// --- TokenUsage tests ---

func TestTokenUsage_Add(t *testing.T) {
	u := NewTokenUsage()
	u.Add("model-a", 100, 50, 10, 20)
	u.Add("model-a", 200, 100, 30, 40)
	u.Add("model-b", 300, 150, 0, 0)

	assert.Len(t, u.Models, 2)
	assert.Equal(t, int64(300), u.Models["model-a"].InputTokens)
	assert.Equal(t, int64(150), u.Models["model-a"].OutputTokens)
	assert.Equal(t, int64(40), u.Models["model-a"].CacheCreationTokens)
	assert.Equal(t, int64(60), u.Models["model-a"].CacheReadTokens)
	assert.Equal(t, int64(300), u.Models["model-b"].InputTokens)
}

func TestTokenUsage_IsEmpty(t *testing.T) {
	u := NewTokenUsage()
	assert.True(t, u.IsEmpty())
	u.Add("m", 1, 0, 0, 0)
	assert.False(t, u.IsEmpty())
}

func TestTokenUsage_Sorted(t *testing.T) {
	u := NewTokenUsage()
	u.Add("z-model", 1, 0, 0, 0)
	u.Add("a-model", 2, 0, 0, 0)
	u.Add("m-model", 3, 0, 0, 0)

	sorted := u.Sorted()
	require.Len(t, sorted, 3)
	assert.Equal(t, "a-model", sorted[0].Model)
	assert.Equal(t, "m-model", sorted[1].Model)
	assert.Equal(t, "z-model", sorted[2].Model)
}

// --- Claude parser tests ---

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

	// Test the internal file parser directly to avoid HOME directory interference
	usage := NewTokenUsage()
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

// setupClaudeHomeProject creates the HOME-based .claude/projects/{hash}/ structure
// that mirrors how Claude Code actually stores session data.
// Returns (home, sandboxPath) for use in tests.
func setupClaudeHomeProject(t *testing.T, sandboxPath string) string {
	t.Helper()
	home := t.TempDir()
	setHome(t, home)

	// Resolve symlinks to match what claudePathHash does
	resolved, err := filepath.EvalSymlinks(sandboxPath)
	require.NoError(t, err)

	hash := claudePathHash(resolved)
	projectDir := filepath.Join(home, ".claude", "projects", hash)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	return projectDir
}

func TestClaudeParser_Parse_HomeBasedPath(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	parser := &ClaudeParser{}
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

	// Setup project dir based on workspace/ path
	projectDir := setupClaudeHomeProject(t, wsDir)

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":200,"output_tokens":100,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	parser := &ClaudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 1)
	assert.Equal(t, int64(200), usage.Models["claude-sonnet-4-20250514"].InputTokens)
}

func TestClaudeParser_Parse_SubagentFiles(t *testing.T) {
	sandbox := t.TempDir()
	projectDir := setupClaudeHomeProject(t, sandbox)

	// Main session
	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(jsonl), 0o644))

	// Subagent session in nested directory
	subagentDir := filepath.Join(projectDir, "subagents", "abc123")
	require.NoError(t, os.MkdirAll(subagentDir, 0o755))
	subagentJSONL := `{"type":"assistant","message":{"model":"claude-haiku-3-20250514","usage":{"input_tokens":300,"output_tokens":150,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(subagentDir, "subagent.jsonl"), []byte(subagentJSONL), 0o644))

	parser := &ClaudeParser{}
	usage, err := parser.Parse(sandbox, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 2, "should find both main and subagent sessions")
	assert.Equal(t, int64(100), usage.Models["claude-sonnet-4-20250514"].InputTokens)
	assert.Equal(t, int64(300), usage.Models["claude-haiku-3-20250514"].InputTokens)
}

func TestClaudeParser_NoFiles(t *testing.T) {
	dir := t.TempDir()
	setHome(t, dir) // Override HOME to avoid scanning real files
	parser := &ClaudeParser{}
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

	// Set file mtime to the past
	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(filePath, pastTime, pastTime))

	parser := &ClaudeParser{}
	// podStartedAt is recent, so old file should be skipped
	usage, err := parser.Parse(sandbox, time.Now().Add(-1*time.Minute))
	require.NoError(t, err)
	assert.Nil(t, usage, "old session file should be skipped")
}

func TestClaudePathHash(t *testing.T) {
	// Unix-style paths (cross-platform: filepath.ToSlash is a no-op on these)
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

	// Windows-style paths (filepath.ToSlash converts \ to / on all platforms)
	assert.Equal(t, "C-Users-test", claudePathHash(`C:\Users\test`))
	assert.Equal(t, "D-workspace", claudePathHash(`D:\workspace`))
}

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
		agentType string
		wantNil   bool
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
		{"Claude", false},  // case insensitive
		{"AIDER", false},
	}

	for _, tt := range tests {
		parser := GetParser(tt.agentType)
		if tt.wantNil {
			assert.Nil(t, parser, "GetParser(%q) should be nil", tt.agentType)
		} else {
			assert.NotNil(t, parser, "GetParser(%q) should not be nil", tt.agentType)
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
