package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupCompositeTestDB creates a minimal SQLite DB for CompositeProvider tests.
func setupCompositeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	db.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
		id INTEGER PRIMARY KEY,
		slug TEXT,
		name TEXT,
		launch_command TEXT,
		description TEXT,
		is_active INTEGER DEFAULT 1,
		config_schema TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec("INSERT INTO agent_types (id, slug, name, launch_command) VALUES (1, 'claude-code', 'Claude Code', 'claude')")

	db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_configs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		agent_type_id INTEGER NOT NULL,
		config_values TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_credential_profiles (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		agent_type_id INTEGER NOT NULL,
		name TEXT,
		description TEXT,
		is_runner_host INTEGER DEFAULT 0,
		credentials_encrypted TEXT DEFAULT '{}',
		is_default INTEGER DEFAULT 0,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func TestNewCompositeProvider(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)
	require.NotNil(t, provider)

	// Verify it implements AgentConfigProvider
	var _ AgentConfigProvider = provider
}

func TestCompositeProvider_GetAgentType_Found(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)

	at, err := provider.GetAgentType(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "claude-code", at.Slug)
	assert.Equal(t, "Claude Code", at.Name)
}

func TestCompositeProvider_GetAgentType_NotFound(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)

	_, err := provider.GetAgentType(context.Background(), 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentTypeNotFound))
}

func TestCompositeProvider_GetUserEffectiveConfig(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)

	overrides := agent.ConfigValues{"model": "opus"}
	result := provider.GetUserEffectiveConfig(context.Background(), 1, 1, overrides)
	// With no stored config, overrides should be reflected
	assert.Equal(t, "opus", result["model"])
}

func TestCompositeProvider_GetEffectiveCredentialsForPod_RunnerHost(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)

	// With nil profileID (runner host mode), should return empty creds, isRunnerHost=true
	creds, isRunnerHost, err := provider.GetEffectiveCredentialsForPod(context.Background(), 1, 1, nil)
	require.NoError(t, err)
	assert.True(t, isRunnerHost)
	assert.Empty(t, creds)
}

func TestCompositeProvider_GetEffectiveCredentialsForPod_ProfileNotFound(t *testing.T) {
	db := setupCompositeTestDB(t)
	agentTypeSvc := NewAgentTypeService(db)
	credSvc := NewCredentialProfileService(db, agentTypeSvc)
	configSvc := NewUserConfigService(db, agentTypeSvc)

	provider := NewCompositeProvider(agentTypeSvc, credSvc, configSvc)

	profileID := int64(999)
	_, _, err := provider.GetEffectiveCredentialsForPod(context.Background(), 1, 1, &profileID)
	require.Error(t, err)
}
