package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompositeProvider(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	credSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	configSvc := newTestUserConfigService(db, agentSvc)

	provider := NewCompositeProvider(agentSvc, credSvc, configSvc)
	require.NotNil(t, provider)

	// Verify it implements AgentConfigProvider
	var _ AgentConfigProvider = provider
}

func TestCompositeProvider_GetAgent_Found(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	credSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	configSvc := newTestUserConfigService(db, agentSvc)

	provider := NewCompositeProvider(agentSvc, credSvc, configSvc)

	at, err := provider.GetAgent(context.Background(), "claude-code")
	require.NoError(t, err)
	assert.Equal(t, "claude-code", at.Slug)
	assert.Equal(t, "Claude Code", at.Name)
}

func TestCompositeProvider_GetAgent_NotFound(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	credSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	configSvc := newTestUserConfigService(db, agentSvc)

	provider := NewCompositeProvider(agentSvc, credSvc, configSvc)

	_, err := provider.GetAgent(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentNotFound))
}

func TestCompositeProvider_GetEffectiveCredentialsForPod_RunnerHost(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	credSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	configSvc := newTestUserConfigService(db, agentSvc)

	provider := NewCompositeProvider(agentSvc, credSvc, configSvc)

	// With nil profileID (runner host mode), should return empty creds, isRunnerHost=true
	creds, isRunnerHost, err := provider.GetEffectiveCredentialsForPod(context.Background(), 1, "claude-code", nil)
	require.NoError(t, err)
	assert.True(t, isRunnerHost)
	assert.Empty(t, creds)
}

func TestCompositeProvider_GetEffectiveCredentialsForPod_ProfileNotFound(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	credSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	configSvc := newTestUserConfigService(db, agentSvc)

	provider := NewCompositeProvider(agentSvc, credSvc, configSvc)

	profileID := int64(999)
	_, _, err := provider.GetEffectiveCredentialsForPod(context.Background(), 1, "claude-code", &profileID)
	require.Error(t, err)
}
