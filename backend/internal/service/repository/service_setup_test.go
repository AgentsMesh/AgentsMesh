package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
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
			slug TEXT NOT NULL,
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

	// Partial unique index: only active (non-deleted) rows are constrained.
	// Mirrors production schema (migration 000081).
	err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS repositories_org_provider_path_unique
		ON repositories (organization_id, provider_type, provider_base_url, slug)
		WHERE deleted_at IS NULL
	`).Error
	if err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	// Create loops table (referenced by Delete/HardDelete for application-level RESTRICT check)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			repository_id INTEGER,
			runner_id INTEGER,
			custom_agent_slug INTEGER
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create loops table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	service := NewService(repo)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	return NewService(repo), db
}

func TestErrorVariables(t *testing.T) {
	if ErrRepositoryNotFound.Error() != "repository not found" {
		t.Errorf("unexpected error message: %s", ErrRepositoryNotFound.Error())
	}
	if ErrRepositoryExists.Error() != "repository already exists" {
		t.Errorf("unexpected error message: %s", ErrRepositoryExists.Error())
	}
}
