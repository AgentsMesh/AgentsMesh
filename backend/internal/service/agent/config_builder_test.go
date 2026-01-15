package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
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

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
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
		is_builtin INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`).Error; err != nil {
		t.Fatalf("Failed to create agent_types table: %v", err)
	}

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_type_id INTEGER NOT NULL,
		config_values BLOB NOT NULL DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, agent_type_id)
	)`).Error; err != nil {
		t.Fatalf("Failed to create user_agent_configs table: %v", err)
	}

	if err := db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_credential_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_type_id INTEGER NOT NULL,
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

	if err := db.Exec(`INSERT INTO agent_types (slug, name, launch_command, config_schema, command_template, credential_schema, is_builtin, is_active)
		VALUES ('claude-code', 'Claude Code', 'claude',
			'{"fields":[{"name":"model","type":"select","default":"opus"},{"name":"perm_mode","type":"select","default":"default"}]}',
			'{"args":[{"args":["--model","{{.config.model}}"]}]}',
			'[{"name":"api_key","type":"secret","env_var":"ANTHROPIC_API_KEY","required":true}]',
			1, 1)`).Error; err != nil {
		t.Fatalf("Failed to seed agent type: %v", err)
	}

	return db
}

// testCompositeProvider combines the three sub-services for testing ConfigBuilder
type testCompositeProvider struct {
	agentTypeSvc  *AgentTypeService
	credentialSvc *CredentialProfileService
	userConfigSvc *UserConfigService
}

func (p *testCompositeProvider) GetAgentType(ctx context.Context, id int64) (*agent.AgentType, error) {
	return p.agentTypeSvc.GetAgentType(ctx, id)
}

func (p *testCompositeProvider) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues {
	return p.userConfigSvc.GetUserEffectiveConfig(ctx, userID, agentTypeID, overrides)
}

func (p *testCompositeProvider) GetEffectiveCredentialsForPod(ctx context.Context, userID, agentTypeID int64, profileID *int64) (agent.EncryptedCredentials, bool, error) {
	return p.credentialSvc.GetEffectiveCredentialsForPod(ctx, userID, agentTypeID, profileID)
}

func createTestProvider(db *gorm.DB) AgentConfigProvider {
	agentTypeSvc := NewAgentTypeService(db)
	credentialSvc := NewCredentialProfileService(db, agentTypeSvc)
	userConfigSvc := NewUserConfigService(db, agentTypeSvc)
	return &testCompositeProvider{
		agentTypeSvc:  agentTypeSvc,
		credentialSvc: credentialSvc,
		userConfigSvc: userConfigSvc,
	}
}

func TestNewConfigBuilder(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	if builder == nil {
		t.Error("NewConfigBuilder returned nil")
	}
	if builder.provider != provider {
		t.Error("provider not set correctly")
	}
}

func TestConfigBuilder_GetConfigSchema(t *testing.T) {
	ctx := context.Background()

	t.Run("get config schema", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		var at agent.AgentType
		db.First(&at)

		schema, err := builder.GetConfigSchema(ctx, at.ID)
		if err != nil {
			t.Fatalf("GetConfigSchema failed: %v", err)
		}

		if schema == nil {
			t.Fatal("GetConfigSchema returned nil")
		}

		if len(schema.Fields) == 0 {
			t.Error("Schema should have fields")
		}
	})

	t.Run("invalid agent type", func(t *testing.T) {
		db := setupConfigBuilderTestDB(t)
		provider := createTestProvider(db)
		builder := NewConfigBuilder(provider)

		_, err := builder.GetConfigSchema(ctx, 99999)
		if err == nil {
			t.Error("Expected error for invalid agent type")
		}
	})
}

func TestConfigBuilder_buildWorkDirConfig(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	tests := []struct {
		name     string
		req      *ConfigBuildRequest
		wantType string
	}{
		{
			name:     "tempdir when no repo or local path",
			req:      &ConfigBuildRequest{},
			wantType: "tempdir",
		},
		{
			name: "worktree when repo URL provided",
			req: &ConfigBuildRequest{
				RepositoryURL: "https://github.com/test/repo.git",
			},
			wantType: "worktree",
		},
		{
			name: "local when local path provided",
			req: &ConfigBuildRequest{
				LocalPath: "/home/user/project",
			},
			wantType: "local",
		},
		{
			name: "worktree takes priority over local",
			req: &ConfigBuildRequest{
				RepositoryURL: "https://github.com/test/repo.git",
				LocalPath:     "/home/user/project",
			},
			wantType: "worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := builder.buildWorkDirConfig(tt.req)
			if config.Type != tt.wantType {
				t.Errorf("buildWorkDirConfig() Type = %s, want %s", config.Type, tt.wantType)
			}
		})
	}
}

func TestConfigBuilder_buildTemplateContext(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	req := &ConfigBuildRequest{
		MCPPort: 19000,
		PodKey:  "test-pod-123",
	}
	config := agent.ConfigValues{
		"model":     "opus",
		"perm_mode": "plan",
	}

	ctx := builder.buildTemplateContext(req, config)

	configMap, ok := ctx["config"].(agent.ConfigValues)
	if !ok {
		t.Fatal("config not found in context")
	}
	if configMap["model"] != "opus" {
		t.Errorf("config.model = %v, want opus", configMap["model"])
	}

	if ctx["mcp_port"] != 19000 {
		t.Errorf("mcp_port = %v, want 19000", ctx["mcp_port"])
	}

	if ctx["pod_key"] != "test-pod-123" {
		t.Errorf("pod_key = %v, want test-pod-123", ctx["pod_key"])
	}

	sandbox, ok := ctx["sandbox"].(map[string]interface{})
	if !ok {
		t.Fatal("sandbox not found in context")
	}
	if sandbox["root_path"] != "{{.sandbox.root_path}}" {
		t.Errorf("sandbox.root_path = %v, want placeholder", sandbox["root_path"])
	}
}

func TestConfigBuilder_buildConfigSchemaResponse(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	tests := []struct {
		name   string
		schema *agent.ConfigSchema
		check  func(*testing.T, *ConfigSchemaResponse)
	}{
		{
			name:   "empty schema",
			schema: &agent.ConfigSchema{},
			check: func(t *testing.T, resp *ConfigSchemaResponse) {
				if len(resp.Fields) != 0 {
					t.Errorf("Fields count = %d, want 0", len(resp.Fields))
				}
			},
		},
		{
			name: "schema with fields",
			schema: &agent.ConfigSchema{
				Fields: []agent.ConfigField{
					{
						Name:     "model",
						Type:     "select",
						Default:  "opus",
						Required: true,
						Options: []agent.FieldOption{
							{Value: "opus"},
							{Value: "sonnet"},
						},
					},
				},
			},
			check: func(t *testing.T, resp *ConfigSchemaResponse) {
				if len(resp.Fields) != 1 {
					t.Fatalf("Fields count = %d, want 1", len(resp.Fields))
				}
				field := resp.Fields[0]
				if field.Name != "model" {
					t.Errorf("Field.Name = %s, want model", field.Name)
				}
				if field.Type != "select" {
					t.Errorf("Field.Type = %s, want select", field.Type)
				}
				if field.Default != "opus" {
					t.Errorf("Field.Default = %v, want opus", field.Default)
				}
				if !field.Required {
					t.Error("Field.Required should be true")
				}
				if len(field.Options) != 2 {
					t.Errorf("Field.Options count = %d, want 2", len(field.Options))
				}
			},
		},
		{
			name: "schema with validation and show_when",
			schema: &agent.ConfigSchema{
				Fields: []agent.ConfigField{
					{
						Name: "custom_model",
						Type: "string",
						Validation: &agent.Validation{
							MinLength: intPtr(1),
							MaxLength: intPtr(100),
						},
						ShowWhen: &agent.Condition{
							Field:    "use_custom",
							Operator: "eq",
							Value:    true,
						},
					},
				},
			},
			check: func(t *testing.T, resp *ConfigSchemaResponse) {
				if len(resp.Fields) != 1 {
					t.Fatalf("Fields count = %d, want 1", len(resp.Fields))
				}
				field := resp.Fields[0]
				if field.Validation == nil {
					t.Error("Field.Validation should not be nil")
				}
				if field.ShowWhen == nil {
					t.Error("Field.ShowWhen should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := builder.buildConfigSchemaResponse(tt.schema)
			tt.check(t, resp)
		})
	}
}

func intPtr(i int) *int {
	return &i
}
