package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFromAgentfile_NormalMode(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active, agentfile_source)
		VALUES ('claude-code', 'Claude Code', 'claude', 1, 1, 'AGENT claude
EXECUTABLE claude
MODE pty
PROMPT_POSITION prepend')`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "claude-code",
		PodKey:              "pod-test-1",
		MergedAgentfileSource: "AGENT claude\nMODE acp\nPROMPT_POSITION prepend",
		Prompt:              "Hello",
		MCPPort:             19000,
		Cols:                80,
		Rows:                24,
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "pod-test-1", cmd.PodKey)
	// Eval produces launch_command and interaction_mode from AgentFile
	assert.Equal(t, "claude", cmd.LaunchCommand)
	assert.Equal(t, "acp", cmd.InteractionMode)
	// Prompt is passed as separate fields (Runner handles injection into args)
	assert.Equal(t, "prepend", cmd.PromptPosition)
	assert.Equal(t, "Hello", cmd.Prompt)
	// LaunchArgs should NOT contain prompt (Runner injects based on PromptPosition)
	for _, arg := range cmd.LaunchArgs {
		assert.NotEqual(t, "Hello", arg, "Backend should not inject prompt into LaunchArgs")
	}
}
