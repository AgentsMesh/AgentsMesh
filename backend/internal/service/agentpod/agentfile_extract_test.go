package agentpod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseAgentfileSrc = `AGENT claude
MODE pty
PROMPT_POSITION prepend
CONFIG mcp_enabled BOOL = true
CONFIG model SELECT("", "sonnet", "opus") = ""
CONFIG permission_mode SELECT("default", "plan", "bypassPermissions") = "default"
`

func TestExtractAgentfileOverrides_ModeOverride(t *testing.T) {
	userLayer := `MODE acp`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "acp", ov.Mode)
}

func TestExtractAgentfileOverrides_BranchOverride(t *testing.T) {
	userLayer := `BRANCH "develop"`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "develop", ov.Branch)
}

func TestExtractAgentfileOverrides_PermissionMode(t *testing.T) {
	userLayer := `CONFIG permission_mode = "bypassPermissions"`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "bypassPermissions", ov.PermissionMode)
}

func TestExtractAgentfileOverrides_RepoSlug(t *testing.T) {
	userLayer := `REPO "dev-org/demo-api"`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "dev-org/demo-api", ov.RepoSlug)
}

func TestExtractAgentfileOverrides_Prompt(t *testing.T) {
	userLayer := `PROMPT "fix this bug"`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "fix this bug", ov.Prompt)
}

func TestExtractAgentfileOverrides_CredentialProfile(t *testing.T) {
	userLayer := `CREDENTIAL "my-profile"`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "my-profile", ov.CredentialProfile)
}

func TestExtractAgentfileOverrides_AllOverrides(t *testing.T) {
	userLayer := `MODE acp
CREDENTIAL "my-profile"
PROMPT "fix this bug"
CONFIG permission_mode = "bypassPermissions"
REPO "dev-org/demo-api"
BRANCH "develop"
`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "acp", ov.Mode)
	assert.Equal(t, "my-profile", ov.CredentialProfile)
	assert.Equal(t, "fix this bug", ov.Prompt)
	assert.Equal(t, "bypassPermissions", ov.PermissionMode)
	assert.Equal(t, "dev-org/demo-api", ov.RepoSlug)
	assert.Equal(t, "develop", ov.Branch)
}

func TestExtractAgentfileOverrides_InvalidLayer(t *testing.T) {
	userLayer := `INVALID @@@ not valid syntax`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	assert.Nil(t, ov)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidAgentfileLayer)
}

func TestExtractAgentfileOverrides_EmptyLayer(t *testing.T) {
	userLayer := ""

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	// All overrides should carry the base defaults (MODE pty, permission_mode "default").
	assert.Equal(t, "pty", ov.Mode)
	assert.Equal(t, "default", ov.PermissionMode)
	// Fields absent in the base AgentFile stay empty.
	assert.Empty(t, ov.Branch)
	assert.Empty(t, ov.RepoSlug)
	assert.Empty(t, ov.Prompt)
	assert.Empty(t, ov.CredentialProfile)
}

func TestExtractAgentfileOverrides_MergeCorrectness(t *testing.T) {
	// Base has MODE pty, user layer overrides with MODE acp → acp wins.
	userLayer := `MODE acp`

	ov, err := extractFromAgentfileLayer(baseAgentfileSrc, userLayer, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "acp", ov.Mode, "user layer MODE should override base MODE")
	// Other base values remain intact.
	assert.Equal(t, "default", ov.PermissionMode)
}
