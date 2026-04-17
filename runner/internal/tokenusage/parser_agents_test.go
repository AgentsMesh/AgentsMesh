package tokenusage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/anthropics/agentsmesh/runner/internal/agents/aider"
	_ "github.com/anthropics/agentsmesh/runner/internal/agents/claude"
	_ "github.com/anthropics/agentsmesh/runner/internal/agents/codex"
	_ "github.com/anthropics/agentsmesh/runner/internal/agents/opencode"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

var epoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

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
		{"Claude", false},
		{"AIDER", false},
	}

	for _, tt := range tests {
		parser := tokenusage.GetParser(tt.agent)
		if tt.wantNil {
			assert.Nil(t, parser, "GetParser(%q) should be nil", tt.agent)
		} else {
			assert.NotNil(t, parser, "GetParser(%q) should not be nil", tt.agent)
		}
	}
}

func TestCollect_UnknownAgent(t *testing.T) {
	usage := tokenusage.Collect("unknown-agent", t.TempDir(), epoch)
	assert.Nil(t, usage)
}

func TestCollect_NoData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	usage := tokenusage.Collect("claude", t.TempDir(), epoch)
	assert.Nil(t, usage)
}

func TestCollect_WithData(t *testing.T) {
	sandbox := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	resolved, err := filepath.EvalSymlinks(sandbox)
	require.NoError(t, err)

	hash := claudePathHash(resolved)
	projectDir := filepath.Join(home, ".claude", "projects", hash)
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	jsonl := `{"type":"assistant","message":{"model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}
`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "s.jsonl"), []byte(jsonl), 0o644))

	usage := tokenusage.Collect("claude", sandbox, epoch)
	require.NotNil(t, usage)
	assert.Len(t, usage.Models, 1)
}

func TestIsModifiedAfter(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

	assert.True(t, tokenusage.IsModifiedAfter(file, epoch))
	assert.True(t, tokenusage.IsModifiedAfter(file, time.Now().Add(-24*time.Hour)))
	assert.False(t, tokenusage.IsModifiedAfter(file, time.Now().Add(24*time.Hour)))
	assert.False(t, tokenusage.IsModifiedAfter(filepath.Join(dir, "nonexistent"), epoch))
}

func claudePathHash(resolvedPath string) string {
	var b []byte
	for _, c := range resolvedPath {
		switch c {
		case '/', '\\':
			b = append(b, '-')
		case ':':
		default:
			b = append(b, byte(c))
		}
	}
	return string(b)
}
