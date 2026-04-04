package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupConfigBuilderTestDB(t *testing.T) *gorm.DB {
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	dbFile := fmt.Sprintf("/tmp/test_%s_%d.db", safeName, time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(dbFile)
	})

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL DEFAULT '',
		executable TEXT,
		default_args TEXT,
		config_schema BLOB DEFAULT '{}',
		command_template BLOB DEFAULT '{}',
		files_template BLOB DEFAULT '[]',
		credential_schema BLOB DEFAULT '[]',
		status_detection BLOB,
		agentfile_source TEXT,
		is_builtin INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		t.Fatalf("Failed to create agents table: %v", err)
	}

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_slug TEXT NOT NULL,
		config_values BLOB NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, agent_slug)
	)`).Error; err != nil {
		t.Fatalf("Failed to create user_agent_configs table: %v", err)
	}

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_credential_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		is_runner_host INTEGER NOT NULL DEFAULT 0,
		credentials_encrypted BLOB,
		is_default INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		t.Fatalf("Failed to create user_agent_credential_profiles table: %v", err)
	}

	return db
}

// testCompositeProvider combines sub-services for testing ConfigBuilder
type testCompositeProvider struct {
	agentSvc      *AgentService
	credentialSvc *CredentialProfileService
}

func (p *testCompositeProvider) GetAgent(ctx context.Context, slug string) (*agent.Agent, error) {
	return p.agentSvc.GetAgent(ctx, slug)
}

func (p *testCompositeProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID int64, agentSlug string, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentSlug, profileID)
}

func (p *testCompositeProvider) ResolveCredentialsByName(ctx context.Context, userID int64, agentSlug, profileName string) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.ResolveCredentialsByName(ctx, userID, agentSlug, profileName)
}

func createTestProvider(db *gorm.DB) AgentConfigProvider {
	agentSvc := newTestAgentService(db)
	credentialSvc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	return &testCompositeProvider{
		agentSvc:      agentSvc,
		credentialSvc: credentialSvc,
	}
}

func TestNewConfigBuilder(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	if builder == nil {
		t.Error("NewConfigBuilder returned nil")
	}
}

func TestConfigBuilder_BuildPodCommand_NoAgentFile(t *testing.T) {
	db := setupConfigBuilderTestDB(t)

	// Insert agent without AgentFile
	db.Exec(`INSERT INTO agents (slug, name, launch_command, is_builtin, is_active)
		VALUES ('no-agentfile', 'NoAgentFile', 'test', 1, 1)`)

	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	var at agent.Agent
	db.Where("slug = ?", "no-agentfile").First(&at)

	_, err := builder.BuildPodCommand(context.Background(), &ConfigBuildRequest{
		AgentSlug: at.Slug,
		PodKey:      "pod-1",
	})

	if err == nil {
		t.Error("Expected error for agent without AgentFile")
	}
}

func intPtr(i int) *int {
	return &i
}
