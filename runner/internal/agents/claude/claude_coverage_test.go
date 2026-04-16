package claude

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func TestClose_CallsControlTrackerClose(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, slog.Default())
	tr.Close()
}

func TestRegisterInit_ClaudeTransportFactory(t *testing.T) {
	tr := acp.NewTransport(TransportType, acp.EventCallbacks{}, slog.Default())
	assert.NotNil(t, tr)
}

func TestClaudeParser_Parse_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	parser := &claudeParser{}
	usage, err := parser.Parse(t.TempDir(), time.Time{})
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestClaudeParser_ParseJSONLFile_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "empty_lines.jsonl")
	content := "\n\n{\"type\":\"assistant\",\"message\":{\"model\":\"m\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5}}}\n\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(file, usage)
	require.NoError(t, err)
	assert.Equal(t, int64(10), usage.Models["m"].InputTokens)
}

func TestClaudeParser_ParseJSONLFile_ZeroUsage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "zero.jsonl")
	content := `{"type":"assistant","message":{"model":"m","usage":{"input_tokens":0,"output_tokens":0}}}` + "\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseClaudeJSONLFile(file, usage)
	require.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestClaudeParser_Parse_EvalSymlinksError(t *testing.T) {
	setHome(t, t.TempDir())
	parser := &claudeParser{}
	usage, err := parser.Parse("/nonexistent/path/that/does/not/exist", time.Time{})
	assert.NoError(t, err)
	assert.Nil(t, usage)
}
