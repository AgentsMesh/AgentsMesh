package runner

import (
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/service/agent"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupAgentTestDB creates a test database with agent_types table
func setupAgentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create agent_types table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			launch_command TEXT NOT NULL,
			executable TEXT,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create agent_types table: %v", err)
	}

	return db
}

func TestNewAgentTypeServiceAdapter(t *testing.T) {
	db := setupAgentTestDB(t)
	agentTypeSvc := agent.NewAgentTypeService(db)

	adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

	assert.NotNil(t, adapter)
	assert.Equal(t, agentTypeSvc, adapter.agentTypeSvc)
}

func TestAgentTypeServiceAdapter_GetAgentTypesForRunner(t *testing.T) {
	t.Run("returns empty list when no agent types", func(t *testing.T) {
		db := setupAgentTestDB(t)
		agentTypeSvc := agent.NewAgentTypeService(db)
		adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

		result := adapter.GetAgentTypesForRunner()

		assert.Empty(t, result)
	})

	t.Run("returns agent types correctly", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert some agent types
		db.Exec(`INSERT INTO agent_types (slug, name, launch_command, executable, is_active)
			VALUES ('claude-code', 'Claude Code', 'claude', 'claude', TRUE)`)
		db.Exec(`INSERT INTO agent_types (slug, name, launch_command, executable, is_active)
			VALUES ('aider', 'Aider', 'aider', 'aider', TRUE)`)

		agentTypeSvc := agent.NewAgentTypeService(db)
		adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

		result := adapter.GetAgentTypesForRunner()

		assert.Len(t, result, 2)
		assert.Equal(t, "claude-code", result[0].Slug)
		assert.Equal(t, "Claude Code", result[0].Name)
		assert.Equal(t, "claude", result[0].LaunchCommand)
		assert.Equal(t, "claude", result[0].Executable)
	})

	t.Run("only returns active agent types", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert active and inactive agent types
		db.Exec(`INSERT INTO agent_types (slug, name, launch_command, executable, is_active)
			VALUES ('claude-code', 'Claude Code', 'claude', 'claude', TRUE)`)
		db.Exec(`INSERT INTO agent_types (slug, name, launch_command, executable, is_active)
			VALUES ('disabled-agent', 'Disabled', 'disabled', 'disabled', FALSE)`)

		agentTypeSvc := agent.NewAgentTypeService(db)
		adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

		result := adapter.GetAgentTypesForRunner()

		assert.Len(t, result, 1)
		assert.Equal(t, "claude-code", result[0].Slug)
	})

	t.Run("handles agent without executable", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert agent type without executable
		db.Exec(`INSERT INTO agent_types (slug, name, launch_command, is_active)
			VALUES ('no-exec', 'No Executable', 'custom-cmd', TRUE)`)

		agentTypeSvc := agent.NewAgentTypeService(db)
		adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

		result := adapter.GetAgentTypesForRunner()

		assert.Len(t, result, 1)
		assert.Equal(t, "no-exec", result[0].Slug)
		assert.Equal(t, "", result[0].Executable)
	})
}

func TestAgentTypeServiceAdapter_ImplementsInterface(t *testing.T) {
	db := setupAgentTestDB(t)
	agentTypeSvc := agent.NewAgentTypeService(db)
	adapter := NewAgentTypeServiceAdapter(agentTypeSvc)

	// Verify it implements AgentTypesProvider interface
	var _ AgentTypesProvider = adapter
}
