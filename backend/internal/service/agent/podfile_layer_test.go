package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestBuildUserLayer(t *testing.T) {
	config := agent.ConfigValues{
		"model":           "opus",
		"permission_mode": "plan",
		"mcp_enabled":     true,
	}
	req := &ConfigBuildRequest{
		RepositoryURL:  "https://github.com/org/repo",
		SourceBranch:   "main",
		CredentialType: "oauth",
	}

	layer := buildUserLayer(config, req)

	assert.Contains(t, layer, `CONFIG model = "opus"`)
	assert.Contains(t, layer, `CONFIG permission_mode = "plan"`)
	assert.Contains(t, layer, "CONFIG mcp_enabled = true")
	assert.Contains(t, layer, `REPO "https://github.com/org/repo"`)
	assert.Contains(t, layer, `BRANCH "main"`)
	assert.Contains(t, layer, "GIT_CREDENTIAL oauth")
}

func TestFormatLiteralValue(t *testing.T) {
	assert.Equal(t, `"hello"`, formatLiteralValue("hello"))
	assert.Equal(t, "true", formatLiteralValue(true))
	assert.Equal(t, "false", formatLiteralValue(false))
	assert.Equal(t, "42", formatLiteralValue(float64(42)))
	assert.Equal(t, "3.14", formatLiteralValue(float64(3.14)))
}

func TestFormatLiteralValue_Escaping(t *testing.T) {
	// Double quotes in value must be escaped to prevent PodFile injection
	assert.Equal(t, `"he said \"hello\""`, formatLiteralValue(`he said "hello"`))
	// Backslashes must be escaped
	assert.Equal(t, `"path\\to\\file"`, formatLiteralValue(`path\to\file`))
	// Combined: backslash followed by quote
	assert.Equal(t, `"a\\\"b"`, formatLiteralValue(`a\"b`))
	// Empty string
	assert.Equal(t, `""`, formatLiteralValue(""))
	// Newlines in value (should be preserved as literal \n in the string)
	assert.Equal(t, "\"line1\\nline2\"", formatLiteralValue("line1\nline2"))
}

func TestConfigToStringMap(t *testing.T) {
	config := agent.ConfigValues{
		"model":   "opus",
		"enabled": true,
		"count":   float64(42),
	}
	result := configToStringMap(config)
	assert.Equal(t, "opus", result["model"])
	assert.Equal(t, "true", result["enabled"])
	assert.Equal(t, "42", result["count"])
}
