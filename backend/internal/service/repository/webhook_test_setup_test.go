package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===========================================
// Test Setup Utilities
// ===========================================

func setupWebhookTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
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
			webhook_config TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
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

func createTestWebhookService(t *testing.T) (*WebhookService, *gorm.DB) {
	db := setupWebhookTestDB(t)
	repo := infra.NewGitProviderRepository(db)
	cfg := &config.Config{}
	// Note: userService is nil - tests that need it should mock appropriately
	svc := NewWebhookService(repo, cfg, nil, nil)
	return svc, db
}

func createTestRepository(t *testing.T, db *gorm.DB) *gitprovider.Repository {
	repo := &gitprovider.Repository{
		OrganizationID:  1,
		ProviderType:    "gitlab",
		ProviderBaseURL: "https://gitlab.com",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		DefaultBranch:   "main",
		Visibility:      "organization",
		IsActive:        true,
	}
	if err := db.Create(repo).Error; err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}
	return repo
}
