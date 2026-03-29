package webhooks

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===========================================
// Test Setup for WithRepo handlers
// ===========================================

func setupTestDBForWithRepo(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create repositories table
	db.Exec(`CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		provider_type TEXT NOT NULL,
		provider_base_url TEXT NOT NULL,
		clone_url TEXT,
		http_clone_url TEXT,
		ssh_clone_url TEXT,
		external_id TEXT NOT NULL,
		name TEXT NOT NULL,
		slug TEXT NOT NULL,
		default_branch TEXT DEFAULT 'main',
		ticket_prefix TEXT,
		visibility TEXT DEFAULT 'organization',
		imported_by_user_id INTEGER,
		is_active BOOLEAN DEFAULT TRUE,
		preparation_script TEXT,
		preparation_timeout INTEGER,
		webhook_config TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	)`)

	return db
}

func createTestRouterForWithRepo(t *testing.T, cfg *config.Config) (*WebhookRouter, *gorm.DB, *repository.Service) {
	db := setupTestDBForWithRepo(t)
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	repoRepo := infra.NewGitProviderRepository(db)
	repoSvc := repository.NewService(repoRepo)

	return &WebhookRouter{
		db:          db,
		cfg:         cfg,
		logger:      logger,
		registry:    registry,
		repoService: repoSvc,
	}, db, repoSvc
}
