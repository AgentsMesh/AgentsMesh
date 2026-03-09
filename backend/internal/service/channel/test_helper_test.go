package channel

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	db.Exec(`CREATE TABLE IF NOT EXISTS channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		document TEXT,
		repository_id INTEGER,
		ticket_id INTEGER,
		created_by_pod TEXT,
		created_by_user_id INTEGER,
		is_archived INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		sender_pod TEXT,
		sender_user_id INTEGER,
		message_type TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		edited_at DATETIME,
		is_deleted INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS pod_bindings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		initiator_pod TEXT NOT NULL,
		target_pod TEXT NOT NULL,
		granted_scopes TEXT DEFAULT '[]',
		pending_scopes TEXT DEFAULT '[]',
		status TEXT NOT NULL DEFAULT 'pending',
		requested_at DATETIME,
		responded_at DATETIME,
		expires_at DATETIME,
		rejection_reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT NOT NULL,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_access (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT,
		user_id INTEGER,
		last_access DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pod_key TEXT NOT NULL UNIQUE,
		organization_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
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
