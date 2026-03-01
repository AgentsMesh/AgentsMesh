package agentpod

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAIProviderTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_ai_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL,
			name TEXT NOT NULL,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			encrypted_credentials TEXT NOT NULL,
			last_used_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_ai_providers table: %v", err)
	}

	return db
}

func TestNewAIProviderService(t *testing.T) {
	db := setupAIProviderTestDB(t)
	service := NewAIProviderService(db, nil) // nil encryptor for development mode

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Fatal("expected service.db to be the provided db")
	}
}
