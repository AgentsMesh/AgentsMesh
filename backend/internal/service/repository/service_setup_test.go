package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS git_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			client_id TEXT,
			client_secret_encrypted TEXT,
			bot_token_encrypted TEXT,
			ssh_key_id INTEGER,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create git_providers table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL DEFAULT 'github',
			provider_base_url TEXT NOT NULL,
			clone_url TEXT,
			http_clone_url TEXT,
			ssh_clone_url TEXT,
			external_id TEXT NOT NULL,
			name TEXT NOT NULL,
			full_path TEXT NOT NULL,
			default_branch TEXT NOT NULL DEFAULT 'main',
			ticket_prefix TEXT,
			visibility TEXT NOT NULL DEFAULT 'organization',
			imported_by_user_id INTEGER,
			preparation_script TEXT,
			preparation_timeout INTEGER DEFAULT 300,
			is_active INTEGER NOT NULL DEFAULT 1,
			webhook_config TEXT,
			deleted_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create repositories table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrRepositoryNotFound.Error() != "repository not found" {
		t.Errorf("unexpected error message: %s", ErrRepositoryNotFound.Error())
	}
	if ErrRepositoryExists.Error() != "repository already exists" {
		t.Errorf("unexpected error message: %s", ErrRepositoryExists.Error())
	}
}
