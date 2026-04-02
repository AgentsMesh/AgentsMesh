package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// integrationProvider wraps the testutil DB and creates a ConfigBuilder provider.
type integrationProvider struct {
	db *gorm.DB
}

func newIntegrationProvider(t *testing.T) (*integrationProvider, *gorm.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	return &integrationProvider{db: db}, db
}

func (ip *integrationProvider) configProvider() AgentConfigProvider {
	agentSvc := NewAgentService(infra.NewAgentRepository(ip.db))
	credentialSvc := NewCredentialProfileService(
		infra.NewCredentialProfileRepository(ip.db),
		agentSvc,
		testEncryptor(),
	)
	userConfigSvc := NewUserConfigService(
		infra.NewUserConfigRepository(ip.db),
		agentSvc,
	)
	return &testCompositeProvider{
		agentSvc:      agentSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
	}
}

func seedAgent(t *testing.T, db *gorm.DB, slug, podfileSrc string) {
	t.Helper()
	testutil.CreateAgent(t, db, slug, "Agent "+slug, podfileSrc)
}

func TestConfigBuilder_BuildBasicCommand(t *testing.T) {
	ip, db := newIntegrationProvider(t)

	podfile := "AGENT test-agent\nEXECUTABLE test-agent\nMODE pty"
	seedAgent(t, db, "test-agent", podfile)

	builder := NewConfigBuilder(ip.configProvider())

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "test-agent",
		PodKey:              "pod-basic-1",
		MergedPodfileSource: podfile,
		MCPPort:             19000,
		Cols:                120,
		Rows:                40,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	assert.Equal(t, "pod-basic-1", cmd.PodKey)
	assert.Equal(t, "test-agent", cmd.LaunchCommand)
	assert.Equal(t, "pty", cmd.InteractionMode)
	assert.Equal(t, int32(120), cmd.Cols)
	assert.Equal(t, int32(40), cmd.Rows)
	assert.Nil(t, cmd.Credentials)
	assert.Nil(t, cmd.SandboxConfig)
}

func TestConfigBuilder_WithCredentials(t *testing.T) {
	ip, db := newIntegrationProvider(t)

	podfile := "AGENT cred-agent\nEXECUTABLE cred-agent\nMODE pty\nENV API_KEY SECRET"
	seedAgent(t, db, "cred-agent", podfile)

	userID := testutil.CreateUser(t, db, "cred-user@test.com", "creduser")

	agentSvc := NewAgentService(infra.NewAgentRepository(db))
	credSvc := NewCredentialProfileService(
		infra.NewCredentialProfileRepository(db),
		agentSvc,
		testEncryptor(),
	)
	profile, err := credSvc.CreateCredentialProfile(context.Background(), userID, &CreateCredentialProfileParams{
		AgentSlug:   "cred-agent",
		Name:        "test-profile",
		IsDefault:   true,
		Credentials: map[string]string{"API_KEY": "sk-test-123"},
	})
	require.NoError(t, err)
	require.NotNil(t, profile)

	builder := NewConfigBuilder(ip.configProvider())

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "cred-agent",
		UserID:              userID,
		PodKey:              "pod-cred-1",
		MergedPodfileSource: podfile,
		MCPPort:             19000,
		Cols:                80,
		Rows:                24,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Credentials injected into env_vars via PodFile eval (ENV API_KEY SECRET)
	require.NotNil(t, cmd.Credentials)
	assert.Contains(t, cmd.Credentials, "API_KEY")
}

func TestConfigBuilder_EvalProducesCorrectOutput(t *testing.T) {
	ip, db := newIntegrationProvider(t)

	podfile := `AGENT merge-agent
EXECUTABLE merge-agent
MODE acp
CONFIG model STRING = "opus"
arg "--model" config.model when config.model != ""
PROMPT_POSITION prepend
`
	seedAgent(t, db, "merge-agent", podfile)

	builder := NewConfigBuilder(ip.configProvider())

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "merge-agent",
		PodKey:              "pod-eval-1",
		MergedPodfileSource: podfile,
		MCPPort:             19000,
		Cols:                80,
		Rows:                24,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Eval produces launch_command from AGENT declaration
	assert.Equal(t, "merge-agent", cmd.LaunchCommand)
	// Eval produces interaction_mode from MODE declaration
	assert.Equal(t, "acp", cmd.InteractionMode)
	// Eval produces launch_args from arg statements (config.model = "opus")
	assert.Contains(t, cmd.LaunchArgs, "--model")
	assert.Contains(t, cmd.LaunchArgs, "opus")
	// Eval produces prompt_position from PROMPT_POSITION declaration
	assert.Equal(t, "prepend", cmd.PromptPosition)

	// Resume fallback: empty MergedPodfileSource → uses base PodFile
	basePodfile := "AGENT merge-agent\nEXECUTABLE merge-agent\nMODE pty"
	seedAgent(t, db, "resume-agent", basePodfile)
	cmdResume, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug: "resume-agent",
		PodKey:    "pod-resume-1",
		MCPPort:   19000,
		Cols:      80,
		Rows:      24,
	})
	require.NoError(t, err)
	assert.Equal(t, "pty", cmdResume.InteractionMode)
}
