package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupAgentTestDB creates a test database with agents table
func setupAgentTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func TestNewAgentServiceAdapter(t *testing.T) {
	db := setupAgentTestDB(t)
	agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))

	adapter := NewAgentServiceAdapter(agentSvc)

	assert.NotNil(t, adapter)
	assert.Equal(t, agentSvc, adapter.agentSvc)
}

func TestAgentServiceAdapter_GetAgentsForRunner(t *testing.T) {
	t.Run("returns empty list when no agents", func(t *testing.T) {
		db := setupAgentTestDB(t)
		agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))
		adapter := NewAgentServiceAdapter(agentSvc)

		result := adapter.GetAgentsForRunner()

		assert.Empty(t, result)
	})

	t.Run("returns agents correctly", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert some agents
		db.Exec(`INSERT INTO agents (slug, name, launch_command, executable, is_active)
			VALUES ('claude-code', 'Claude Code', 'claude', 'claude', TRUE)`)
		db.Exec(`INSERT INTO agents (slug, name, launch_command, executable, is_active)
			VALUES ('aider', 'Aider', 'aider', 'aider', TRUE)`)

		agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))
		adapter := NewAgentServiceAdapter(agentSvc)

		result := adapter.GetAgentsForRunner()

		assert.Len(t, result, 2)
		assert.Equal(t, "claude-code", result[0].Slug)
		assert.Equal(t, "Claude Code", result[0].Name)
		assert.Equal(t, "claude", result[0].LaunchCommand)
		assert.Equal(t, "claude", result[0].Executable)
	})

	t.Run("only returns active agents", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert active and inactive agents
		db.Exec(`INSERT INTO agents (slug, name, launch_command, executable, is_active)
			VALUES ('claude-code', 'Claude Code', 'claude', 'claude', TRUE)`)
		db.Exec(`INSERT INTO agents (slug, name, launch_command, executable, is_active)
			VALUES ('disabled-agent', 'Disabled', 'disabled', 'disabled', FALSE)`)

		agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))
		adapter := NewAgentServiceAdapter(agentSvc)

		result := adapter.GetAgentsForRunner()

		assert.Len(t, result, 1)
		assert.Equal(t, "claude-code", result[0].Slug)
	})

	t.Run("handles agent without executable", func(t *testing.T) {
		db := setupAgentTestDB(t)

		// Insert agent without executable
		db.Exec(`INSERT INTO agents (slug, name, launch_command, is_active)
			VALUES ('no-exec', 'No Executable', 'custom-cmd', TRUE)`)

		agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))
		adapter := NewAgentServiceAdapter(agentSvc)

		result := adapter.GetAgentsForRunner()

		assert.Len(t, result, 1)
		assert.Equal(t, "no-exec", result[0].Slug)
		assert.Equal(t, "", result[0].Executable)
	})
}

func TestAgentServiceAdapter_ImplementsInterface(t *testing.T) {
	db := setupAgentTestDB(t)
	agentSvc := agent.NewAgentService(infra.NewAgentRepository(db))
	adapter := NewAgentServiceAdapter(agentSvc)

	// Verify it implements AgentsProvider interface
	var _ interfaces.AgentsProvider = adapter
}
