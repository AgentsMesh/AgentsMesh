package webhooks

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDBForGit(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func testLoggerForGit() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestRouterForGit(t *testing.T, cfg *config.Config) (*WebhookRouter, *gorm.DB) {
	db := testkit.SetupTestDB(t)
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
