package channel

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testkit.SetupTestDB(t)
}

// newTestLogger creates a logger for testing.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// Helper functions for pointer creation
func intPtr(i int64) *int64 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

// newTestService creates a channel Service backed by an in-memory DB for testing.
func newTestService(db *gorm.DB) *Service {
	return NewService(infra.NewChannelRepository(db))
}
