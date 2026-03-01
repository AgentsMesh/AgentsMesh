package auth

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Mock user for testing
func createMockUser() *user.User {
	name := "Test User"
	return &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Name:     &name,
		IsActive: true,
	}
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			name TEXT,
			avatar_url TEXT,
			password_hash TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_system_admin INTEGER NOT NULL DEFAULT 0,
			last_login_at DATETIME,
			is_email_verified INTEGER NOT NULL DEFAULT 0,
			email_verification_token TEXT,
			email_verification_expires_at DATETIME,
			password_reset_token TEXT,
			password_reset_expires_at DATETIME,
			default_git_credential_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_identities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			provider_user_id TEXT NOT NULL,
			provider_username TEXT,
			access_token_encrypted TEXT,
			refresh_token_encrypted TEXT,
			token_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_identities table: %v", err)
	}

	return db
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
