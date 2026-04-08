package webhooks

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// ===========================================
// Test Setup for WithRepo handlers
// ===========================================

func setupTestDBForWithRepo(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
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
