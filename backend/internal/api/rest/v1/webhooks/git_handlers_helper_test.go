package webhooks

import (
	"log/slog"
	"os"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForGit(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	return db
}

func testLoggerForGit() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestRouterForGit(cfg *config.Config) (*WebhookRouter, *gorm.DB) {
	db := setupTestDBForGitRouter()
	logger := testLoggerForGit()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	return &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}, db
}

func setupTestDBForGitRouter() *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	return db
}
