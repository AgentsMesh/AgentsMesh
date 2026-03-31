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
		Cols:                120,
		Rows:                40,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	assert.Equal(t, "pod-basic-1", cmd.PodKey)
	assert.Equal(t, podfile, cmd.PodfileSource)
	assert.Equal(t, int32(120), cmd.Cols)
	assert.Equal(t, int32(40), cmd.Rows)
	// No credentials → map should be nil
	assert.Nil(t, cmd.Credentials)
	// No sandbox config when no repo is specified
	assert.Nil(t, cmd.SandboxConfig)
}

func TestConfigBuilder_WithCredentials(t *testing.T) {
	ip, db := newIntegrationProvider(t)

	podfile := "AGENT cred-agent\nEXECUTABLE cred-agent\nMODE pty"
	seedAgent(t, db, "cred-agent", podfile)

	// Create a user
	userID := testutil.CreateUser(t, db, "cred-user@test.com", "creduser")

	// Create a credential profile with encrypted credentials
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
		Cols:                80,
		Rows:                24,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Should have credentials populated (encrypted values)
	require.NotNil(t, cmd.Credentials)
	assert.Contains(t, cmd.Credentials, "API_KEY")
	assert.NotEmpty(t, cmd.Credentials["API_KEY"])
}

func TestConfigBuilder_PodfileMerge(t *testing.T) {
	ip, db := newIntegrationProvider(t)

	basePodfile := "AGENT merge-agent\nEXECUTABLE merge-agent\nMODE pty"
	seedAgent(t, db, "merge-agent", basePodfile)

	builder := NewConfigBuilder(ip.configProvider())

	// Simulate merged podfile (orchestrator merges base + layer before calling builder)
	mergedSource := "AGENT merge-agent\nEXECUTABLE merge-agent\nMODE acp\nENV FOO=bar"

	cmd, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "merge-agent",
		PodKey:              "pod-merge-1",
		MergedPodfileSource: mergedSource,
		Cols:                80,
		Rows:                24,
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Builder should use the merged source directly
	assert.Equal(t, mergedSource, cmd.PodfileSource)
	assert.Contains(t, cmd.PodfileSource, "MODE acp")
	assert.Contains(t, cmd.PodfileSource, "ENV FOO=bar")

	// Resume fallback: empty MergedPodfileSource → uses base PodFile
	cmdResume, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug:           "merge-agent",
		PodKey:              "pod-resume-1",
		MergedPodfileSource: "", // empty = resume mode
		Cols:                80,
		Rows:                24,
	})
	require.NoError(t, err)
	require.NotNil(t, cmdResume)
	assert.Contains(t, cmdResume.PodfileSource, "MODE pty")
	assert.NotContains(t, cmdResume.PodfileSource, "MODE acp")
}
