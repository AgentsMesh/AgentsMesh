package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/podfile/merge"
	"github.com/anthropics/agentsmesh/podfile/parser"
	"github.com/anthropics/agentsmesh/podfile/serialize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPodFile_LayerOverride_MCPOff verifies that a user layer with MCP OFF
// disables MCP, so build logic `if mcp.enabled { ... }` is skipped.
func TestPodFile_LayerOverride_MCPOff(t *testing.T) {
	base := realClaudePodFile
	layer := "MCP OFF\n"

	merged := mergeAndEval(t, base, layer, map[string]string{}, nil)

	// MCP OFF → no plugin files created
	for _, f := range merged.FilesToCreate {
		if !f.IsDirectory {
			assert.NotContains(t, f.Path, ".mcp.json", "MCP OFF should skip file creation")
		}
	}
	assert.NotContains(t, merged.LaunchArgs, "--plugin-dir")
}

// TestPodFile_LayerOverride_ConfigDefault verifies that a user layer can
// override CONFIG defaults without redeclaring the full CONFIG.
func TestPodFile_LayerOverride_ConfigDefault(t *testing.T) {
	base := realCodexPodFile

	// Layer overrides approval_mode default (normally "untrusted")
	merged := mergeAndEval(t, base, "", map[string]string{"approval_mode": "never"}, nil)

	assert.Contains(t, merged.LaunchArgs, "--ask-for-approval")
	assert.Contains(t, merged.LaunchArgs, "never")
}

// TestPodFile_LayerOverride_RemoveEnv verifies REMOVE ENV in a layer.
func TestPodFile_LayerOverride_RemoveEnv(t *testing.T) {
	base := realClaudePodFile
	layer := "REMOVE ENV \"ANTHROPIC_AUTH_TOKEN\"\n"

	merged := mergeAndEval(t, base, layer, map[string]string{},
		map[string]string{"ANTHROPIC_API_KEY": "sk-1", "ANTHROPIC_AUTH_TOKEN": "tok-2"})

	// ANTHROPIC_API_KEY still injected, AUTH_TOKEN removed
	assert.Equal(t, "sk-1", merged.EnvVars["ANTHROPIC_API_KEY"])
	_, hasToken := merged.EnvVars["ANTHROPIC_AUTH_TOKEN"]
	assert.False(t, hasToken, "REMOVE ENV should prevent injection")
}

// TestPodFile_LayerOverride_AddEnv verifies adding an ENV via layer.
func TestPodFile_LayerOverride_AddEnv(t *testing.T) {
	base := realGeminiPodFile
	layer := "ENV CUSTOM_FLAG = \"enabled\"\n"

	merged := mergeAndEval(t, base, layer, map[string]string{}, nil)

	assert.Equal(t, "enabled", merged.EnvVars["CUSTOM_FLAG"])
}

// TestPodFile_Gemini_SandboxMode verifies sandbox_mode=true adds --sandbox.
func TestPodFile_Gemini_SandboxMode(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realGeminiPodFile
	cmd.ConfigValues = map[string]string{"sandbox_mode": "true"}

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)
	assert.Contains(t, result.LaunchArgs, "--sandbox")
}

// TestPodFile_Claude_PlanMode verifies permission_mode="plan" path.
func TestPodFile_Claude_PlanMode(t *testing.T) {
	cmd := mcpCmd()
	cmd.PodfileSource = realClaudePodFile
	cmd.ConfigValues = map[string]string{"permission_mode": "plan"}

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)

	assert.Contains(t, result.LaunchArgs, "--permission-mode")
	assert.Contains(t, result.LaunchArgs, "plan")
	assert.NotContains(t, result.LaunchArgs, "--dangerously-skip-permissions")
}

// mergeAndEval merges a base PodFile with a layer string, then evaluates.
func mergeAndEval(t *testing.T, base, layer string, config map[string]string, creds map[string]string) *PodFileResult {
	t.Helper()

	baseProg, errs := parser.Parse(base)
	require.Empty(t, errs)

	if layer != "" {
		layerProg, errs := parser.Parse(layer)
		require.Empty(t, errs)
		merge.Merge(baseProg, layerProg)
	}

	// Re-serialize to source for ExecutePodFile
	// (ExecutePodFile re-parses, but we need the merged source)
	// Alternative: call eval directly. Let's use the serialize package.
	// Actually, simpler: build the cmd with merged source via serialize.
	// But serialize is in podfile/serialize — let's just eval directly.

	cmd := mcpCmd()
	cmd.ConfigValues = config
	cmd.Credentials = creds
	if cmd.Credentials == nil {
		cmd.Credentials = map[string]string{}
	}

	// Use serialize to get merged source
	src := serialize.Serialize(baseProg)
	cmd.PodfileSource = src

	result, err := ExecutePodFile(cmd, "/tmp/sb", "/tmp/sb/ws")
	require.NoError(t, err)
	return result
}
