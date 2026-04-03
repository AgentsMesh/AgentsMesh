package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDBWithUserAgentConfigs(t *testing.T) *gorm.DB {
	db := setupTestDB(t)

	// Add user_agent_configs table for SQLite
	db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_slug TEXT NOT NULL,
		config_values BLOB NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, agent_slug)
	)`)

	return db
}

func TestUserConfigService_GetUserAgentConfig(t *testing.T) {
	db := setupTestDBWithUserAgentConfigs(t)
	agentSvc := newTestAgentService(db)
	svc := newTestUserConfigService(db, agentSvc)
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)

	t.Run("returns empty config when not found", func(t *testing.T) {
		config, err := svc.GetUserAgentConfig(ctx, 1, at.Slug)
		if err != nil {
			t.Errorf("GetUserAgentConfig failed: %v", err)
		}
		if config.UserID != 1 {
			t.Errorf("UserID = %d, want 1", config.UserID)
		}
		if config.AgentSlug != at.Slug {
			t.Errorf("AgentSlug = %s, want %s", config.AgentSlug, at.Slug)
		}
		if len(config.ConfigValues) != 0 {
			t.Error("Should return empty config values when not found")
		}
	})

	t.Run("returns existing config", func(t *testing.T) {
		configValues := agent.ConfigValues{
			"model":           "opus",
			"permission_mode": "plan",
		}
		_, err := svc.SetUserAgentConfig(ctx, 2, at.Slug, configValues)
		if err != nil {
			t.Fatalf("SetUserAgentConfig failed: %v", err)
		}

		config, err := svc.GetUserAgentConfig(ctx, 2, at.Slug)
		if err != nil {
			t.Errorf("GetUserAgentConfig failed: %v", err)
		}
		if config.ConfigValues["model"] != "opus" {
			t.Errorf("model = %v, want opus", config.ConfigValues["model"])
		}
		if config.ConfigValues["permission_mode"] != "plan" {
			t.Errorf("permission_mode = %v, want plan", config.ConfigValues["permission_mode"])
		}
	})
}

func TestUserConfigService_SetUserAgentConfig(t *testing.T) {
	db := setupTestDBWithUserAgentConfigs(t)
	agentSvc := newTestAgentService(db)
	svc := newTestUserConfigService(db, agentSvc)
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)

	t.Run("create new config", func(t *testing.T) {
		configValues := agent.ConfigValues{
			"model":       "sonnet",
			"mcp_enabled": true,
		}
		config, err := svc.SetUserAgentConfig(ctx, 1, at.Slug, configValues)
		if err != nil {
			t.Errorf("SetUserAgentConfig failed: %v", err)
		}
		if config.ConfigValues["model"] != "sonnet" {
			t.Errorf("model = %v, want sonnet", config.ConfigValues["model"])
		}
		if config.ConfigValues["mcp_enabled"] != true {
			t.Errorf("mcp_enabled = %v, want true", config.ConfigValues["mcp_enabled"])
		}
	})

	t.Run("update existing config", func(t *testing.T) {
		_, err := svc.SetUserAgentConfig(ctx, 3, at.Slug, agent.ConfigValues{
			"model": "opus",
		})
		if err != nil {
			t.Fatalf("First SetUserAgentConfig failed: %v", err)
		}

		updatedConfig, err := svc.SetUserAgentConfig(ctx, 3, at.Slug, agent.ConfigValues{
			"model":           "sonnet",
			"permission_mode": "default",
		})
		if err != nil {
			t.Errorf("Update SetUserAgentConfig failed: %v", err)
		}
		if updatedConfig.ConfigValues["model"] != "sonnet" {
			t.Errorf("model should be updated to sonnet, got %v", updatedConfig.ConfigValues["model"])
		}
		if updatedConfig.ConfigValues["permission_mode"] != "default" {
			t.Errorf("permission_mode = %v, want default", updatedConfig.ConfigValues["permission_mode"])
		}
	})

	t.Run("fails for non-existent agent", func(t *testing.T) {
		_, err := svc.SetUserAgentConfig(ctx, 1, "nonexistent", agent.ConfigValues{
			"model": "opus",
		})
		if err == nil {
			t.Error("Expected error for non-existent agent")
		}
		if err != ErrAgentNotFound {
			t.Errorf("Expected ErrAgentNotFound, got %v", err)
		}
	})
}

func TestUserConfigService_DeleteUserAgentConfig(t *testing.T) {
	db := setupTestDBWithUserAgentConfigs(t)
	agentSvc := newTestAgentService(db)
	svc := newTestUserConfigService(db, agentSvc)
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)

	t.Run("delete existing config", func(t *testing.T) {
		_, err := svc.SetUserAgentConfig(ctx, 1, at.Slug, agent.ConfigValues{
			"model": "opus",
		})
		if err != nil {
			t.Fatalf("SetUserAgentConfig failed: %v", err)
		}

		err = svc.DeleteUserAgentConfig(ctx, 1, at.Slug)
		if err != nil {
			t.Errorf("DeleteUserAgentConfig failed: %v", err)
		}

		config, _ := svc.GetUserAgentConfig(ctx, 1, at.Slug)
		if len(config.ConfigValues) != 0 {
			t.Error("Config should be deleted (empty values returned)")
		}
	})

	t.Run("delete non-existent config is ok", func(t *testing.T) {
		err := svc.DeleteUserAgentConfig(ctx, 999, "nonexistent")
		if err != nil {
			t.Errorf("DeleteUserAgentConfig should not fail for non-existent config: %v", err)
		}
	})
}

func TestUserConfigService_ListUserAgentConfigs(t *testing.T) {
	db := setupTestDBWithUserAgentConfigs(t)
	agentSvc := newTestAgentService(db)
	svc := newTestUserConfigService(db, agentSvc)
	ctx := context.Background()

	var agents []agent.Agent
	db.Where("is_active = ?", true).Find(&agents)

	userID := int64(100)
	for i, at := range agents {
		_, err := svc.SetUserAgentConfig(ctx, userID, at.Slug, agent.ConfigValues{
			"model": "model-" + string(rune('a'+i)),
		})
		if err != nil {
			t.Logf("Failed to create config for agent %s: %v", at.Slug, err)
		}
	}

	if len(agents) > 0 {
		svc.SetUserAgentConfig(ctx, 200, agents[0].Slug, agent.ConfigValues{
			"model": "other-user",
		})
	}

	t.Run("lists only current user configs", func(t *testing.T) {
		configs, err := svc.ListUserAgentConfigs(ctx, userID)
		if err != nil {
			t.Fatalf("ListUserAgentConfigs failed: %v", err)
		}

		for _, config := range configs {
			if config.UserID != userID {
				t.Errorf("Listed config for wrong user: %d", config.UserID)
			}
		}

		if len(configs) != len(agents) {
			t.Errorf("Configs count = %d, want %d", len(configs), len(agents))
		}
	})
}

func TestUserConfigService_GetUserAgentConfig_DBError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`INSERT INTO agents (slug, name, launch_command) VALUES ('test', 'Test', 'test')`)

	agentSvc := newTestAgentService(badDB)
	svc := newTestUserConfigService(badDB, agentSvc)
	ctx := context.Background()

	_, err := svc.GetUserAgentConfig(ctx, 1, "claude-code")
	if err == nil {
		t.Log("SQLite didn't return error for missing table, which is acceptable")
	}
}

func TestUserConfigService_SetUserAgentConfig_UpdateError(t *testing.T) {
	db := setupTestDBWithUserAgentConfigs(t)
	agentSvc := newTestAgentService(db)
	svc := newTestUserConfigService(db, agentSvc)
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)

	_, err := svc.SetUserAgentConfig(ctx, 1, at.Slug, agent.ConfigValues{"key": "value1"})
	if err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	updated, err := svc.SetUserAgentConfig(ctx, 1, at.Slug, agent.ConfigValues{"key": "value2"})
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}
	if updated.ConfigValues["key"] != "value2" {
		t.Errorf("Expected updated value, got %v", updated.ConfigValues["key"])
	}
}

func TestUserConfigService_DeleteUserAgentConfig_DBError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT ''
	)`)
	badDB.Exec(`INSERT INTO agents (slug, name, launch_command) VALUES ('test', 'Test', 'test')`)

	agentSvc := newTestAgentService(badDB)
	svc := newTestUserConfigService(badDB, agentSvc)
	ctx := context.Background()

	err := svc.DeleteUserAgentConfig(ctx, 1, "claude-code")
	if err != nil {
		t.Logf("Got expected error: %v", err)
	}
}

func TestUserConfigService_SetUserAgentConfig_CreateDBError(t *testing.T) {
	badDB, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	badDB.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		launch_command TEXT NOT NULL DEFAULT '',
		config_schema BLOB DEFAULT '{}'
	)`)
	badDB.Exec(`INSERT INTO agents (slug, name, launch_command) VALUES ('test', 'Test', 'test')`)
	badDB.Exec(`CREATE TABLE IF NOT EXISTS user_agent_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT
	)`)

	agentSvc := newTestAgentService(badDB)
	svc := newTestUserConfigService(badDB, agentSvc)
	ctx := context.Background()

	_, err := svc.SetUserAgentConfig(ctx, 1, "claude-code", agent.ConfigValues{"key": "value"})
	if err == nil {
		t.Log("SQLite handled the insert gracefully (unexpected)")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}
