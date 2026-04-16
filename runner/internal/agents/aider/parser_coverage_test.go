package aider

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func TestAiderParser_Parse_FileInRootSandbox(t *testing.T) {
	dir := t.TempDir()

	history := "> Tokens: 500 sent, 200 received\n"
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".aider.chat.history.md"), []byte(history), 0o644,
	))

	parser := &aiderParser{}
	usage, err := parser.Parse(dir, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)

	m := usage.Models["aider-unknown"]
	require.NotNil(t, m)
	assert.Equal(t, int64(500), m.InputTokens)
	assert.Equal(t, int64(200), m.OutputTokens)
}

func TestAiderParser_Parse_BothLocations(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(wsDir, ".aider.chat.history.md"),
		[]byte("> Tokens: 100 sent, 50 received\n"), 0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".aider.chat.history.md"),
		[]byte("> Tokens: 200 sent, 100 received\n"), 0o644,
	))

	parser := &aiderParser{}
	usage, err := parser.Parse(dir, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)

	m := usage.Models["aider-unknown"]
	require.NotNil(t, m)
	assert.Equal(t, int64(300), m.InputTokens)
	assert.Equal(t, int64(150), m.OutputTokens)
}

func TestParseAiderHistoryFile_NotExist(t *testing.T) {
	usage := tokenusage.NewTokenUsage()
	err := parseAiderHistoryFile("/nonexistent/path/file.md", usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestParseAiderHistoryFile_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".aider.chat.history.md")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))
	require.NoError(t, os.Chmod(file, 0o000))
	t.Cleanup(func() { os.Chmod(file, 0o644) })

	usage := tokenusage.NewTokenUsage()
	err := parseAiderHistoryFile(file, usage)
	assert.Error(t, err)
}

func TestParseAiderHistoryFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".aider.chat.history.md")
	require.NoError(t, os.WriteFile(file, []byte(""), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseAiderHistoryFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestParseAiderHistoryFile_NoSentReceived(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, ".aider.chat.history.md")
	content := "> Tokens: 0 sent, 0 received, 100 cache write\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseAiderHistoryFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestAiderParser_Parse_SkipsOldFiles(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "workspace")
	require.NoError(t, os.MkdirAll(wsDir, 0o755))

	file := filepath.Join(wsDir, ".aider.chat.history.md")
	require.NoError(t, os.WriteFile(
		file, []byte("> Tokens: 500 sent, 200 received\n"), 0o644,
	))

	pastTime := time.Now().Add(-1 * time.Hour)
	require.NoError(t, os.Chtimes(file, pastTime, pastTime))

	parser := &aiderParser{}
	usage, err := parser.Parse(dir, time.Now().Add(-1*time.Minute))
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestParseAiderHistoryFile_TokensOnlyCache(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "history.md")
	content := "> Tokens: 45k cache write, 123k cache read\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))

	usage := tokenusage.NewTokenUsage()
	err := parseAiderHistoryFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestParseTokenValue_UppercaseSuffixes(t *testing.T) {
	assert.Equal(t, int64(5000), parseTokenValue("5", "K"))
	assert.Equal(t, int64(2000000), parseTokenValue("2", "M"))
}
