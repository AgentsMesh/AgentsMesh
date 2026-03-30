package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFromPodFile_NormalMode(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	// Insert agent with PodFile source
	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, podfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE pty
PROMPT_POSITION prepend')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "claude-code",
		PodKey:              "pod-test-1",
		MergedPodfileSource: "AGENT claude\nMODE acp\nPROMPT_POSITION prepend",
		InitialPrompt:       "Hello",
		Cols:                80,
		Rows:                24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "pod-test-1", cmd.PodKey)
	assert.Equal(t, "AGENT claude\nMODE acp\nPROMPT_POSITION prepend", cmd.PodfileSource)
	assert.Equal(t, "Hello", cmd.InitialPrompt)
	assert.Equal(t, int32(80), cmd.Cols)
}

func TestBuildFromPodFile_ResumeFallback(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, podfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE pty
PROMPT_POSITION prepend')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	// Resume mode: MergedPodfileSource is empty, should fall back to base PodFile
	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "claude-code",
		PodKey:              "pod-resume-1",
		MergedPodfileSource: "", // empty = resume mode
		Cols:                80,
		Rows:                24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	// Should use agent's base PodFile source (not empty)
	assert.Contains(t, cmd.PodfileSource, "AGENT claude")
	assert.Contains(t, cmd.PodfileSource, "MODE pty")
}

func TestConfigToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "string value",
			input:    map[string]interface{}{"key": "value"},
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "bool value",
			input:    map[string]interface{}{"enabled": true},
			expected: map[string]string{"enabled": "true"},
		},
		{
			name:     "float64 value",
			input:    map[string]interface{}{"temp": 0.7},
			expected: map[string]string{"temp": "0.7"},
		},
		{
			name:     "mixed types",
			input:    map[string]interface{}{"s": "hello", "b": false, "n": float64(42)},
			expected: map[string]string{"s": "hello", "b": "false", "n": "42"},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configToStringMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
