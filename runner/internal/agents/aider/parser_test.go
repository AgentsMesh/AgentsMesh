package aider

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var epoch = time.Time{}

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

	parser := &aiderParser{}
	usage, err := parser.Parse(dir, epoch)
	require.NoError(t, err)
	require.NotNil(t, usage)

	m := usage.Models["aider-unknown"]
	require.NotNil(t, m)
	assert.Equal(t, int64(13234), m.InputTokens)
	assert.Equal(t, int64(3967), m.OutputTokens)
	assert.Equal(t, int64(45000), m.CacheCreationTokens)
	assert.Equal(t, int64(123000), m.CacheReadTokens)
}

func TestAiderParser_NoFile(t *testing.T) {
	parser := &aiderParser{}
	usage, err := parser.Parse(t.TempDir(), epoch)
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
