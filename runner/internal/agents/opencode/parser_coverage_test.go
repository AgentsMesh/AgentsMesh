package opencode

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

func TestOpenCodeParser_Parse_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", "")
	}
	parser := &opencodeParser{}
	usage, err := parser.Parse(t.TempDir(), epoch)
	assert.NoError(t, err)
	assert.Nil(t, usage)
}

func TestOpenCodeParser_ParseFile_Oversized(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "big.json")
	data := make([]byte, maxOpenCodeFileSize+1)
	require.NoError(t, os.WriteFile(file, data, 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestOpenCodeParser_ParseFile_ZeroUsage(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "zero.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"model":"m","usage":{"input_tokens":0,"output_tokens":0}}`), 0644))

	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile(file, usage)
	assert.NoError(t, err)
	assert.True(t, usage.IsEmpty())
}

func TestOpenCodeParser_ParseFile_NonexistentFile(t *testing.T) {
	usage := tokenusage.NewTokenUsage()
	err := parseOpenCodeFile("/nonexistent/file.json", usage)
	assert.Error(t, err)
}
