package agent

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database with all required tables for testing.
// This is the shared helper function used by all service tests in this package.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	db.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		slug TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT,
		executable TEXT,
		launch_command TEXT NOT NULL DEFAULT '',
		default_args TEXT,
		config_schema BLOB DEFAULT '{}',
		command_template BLOB DEFAULT '{}',
		files_template BLOB,
		credential_schema BLOB DEFAULT '[]',
		status_detection BLOB,
		is_builtin INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)


	db.Exec(`CREATE TABLE IF NOT EXISTS user_agent_credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		agent_type_id INTEGER NOT NULL,
		credentials_encrypted BLOB,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS custom_agent_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		slug TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		launch_command TEXT NOT NULL,
		default_args TEXT,
		credential_schema BLOB DEFAULT '[]',
		status_detection BLOB,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// Seed builtin agent types using BLOB for credential_schema
	db.Exec(`INSERT INTO agent_types (slug, name, description, executable, launch_command, credential_schema, is_builtin, is_active)
		VALUES ('claude-code', 'Claude Code', 'Claude Code agent', 'claude', 'claude', X'5B5D', 1, 1)`)
	db.Exec(`INSERT INTO agent_types (slug, name, description, executable, launch_command, credential_schema, is_builtin, is_active)
		VALUES ('codex', 'Codex', 'Codex agent', 'codex', 'codex', X'5B5D', 1, 1)`)
	db.Exec(`INSERT INTO agent_types (slug, name, description, executable, launch_command, credential_schema, is_builtin, is_active)
		VALUES ('inactive-agent', 'Inactive', 'Inactive agent', 'inactive', 'inactive', X'5B5D', 1, 0)`)

	return db
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
		{ErrAgentTypeNotFound, "agent type not found"},
		{ErrAgentSlugExists, "agent type slug already exists"},
		{ErrCredentialsRequired, "required credentials missing"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}
