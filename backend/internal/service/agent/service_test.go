package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with all required tables for testing.
// This is the shared helper function used by all service tests in this package.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testkit.SetupTestDB(t)

	// Seed builtin agents
	db.Exec(`INSERT INTO agents (slug, name, description, launch_command, executable, is_builtin, is_active)
		VALUES ('claude-code', 'Claude Code', 'Claude Code agent', 'claude', 'claude', 1, 1)`)
	db.Exec(`INSERT INTO agents (slug, name, description, launch_command, executable, is_builtin, is_active)
		VALUES ('codex', 'Codex', 'Codex agent', 'codex', 'codex', 1, 1)`)
	db.Exec(`INSERT INTO agents (slug, name, description, launch_command, executable, is_builtin, is_active)
		VALUES ('inactive-agent', 'Inactive', 'Inactive agent', 'inactive', 'inactive', 1, 0)`)

	return db
}

// Test helper functions that wrap *gorm.DB into Repository interfaces via infra layer.
// This keeps the infra import in one place rather than every test file.

func newTestAgentService(db *gorm.DB) *AgentService {
	return NewAgentService(infra.NewAgentRepository(db))
}

func newTestCredentialProfileService(db *gorm.DB, atSvc AgentProvider, enc *crypto.Encryptor) *CredentialProfileService {
	return NewCredentialProfileService(infra.NewCredentialProfileRepository(db), atSvc, enc)
}

func newTestUserConfigService(db *gorm.DB, atSvc AgentProvider) *UserConfigService {
	return NewUserConfigService(infra.NewUserConfigRepository(db), atSvc)
}

func newTestMessageService(db *gorm.DB) *MessageService {
	return NewMessageService(infra.NewAgentMessageRepository(db))
}

// strPtr is a helper function to create a pointer to a string value.
func strPtr(s string) *string {
	return &s
}

// TestErrors verifies that all error constants have the expected message strings.
func TestErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrAgentNotFound, "agent not found"},
		{ErrAgentSlugExists, "agent slug already exists"},
		{ErrCredentialsRequired, "required credentials missing"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}
