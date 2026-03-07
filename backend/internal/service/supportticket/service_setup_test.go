package supportticket

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
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

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS support_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'other',
			status TEXT NOT NULL DEFAULT 'open',
			priority TEXT NOT NULL DEFAULT 'medium',
			assigned_admin_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create support_tickets table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS support_ticket_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			is_admin_reply INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create support_ticket_messages table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS support_ticket_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			message_id INTEGER,
			uploader_id INTEGER NOT NULL,
			original_name TEXT NOT NULL,
			storage_key TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			size INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create support_ticket_attachments table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			username TEXT NOT NULL,
			name TEXT,
			avatar_url TEXT,
			password_hash TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_system_admin INTEGER NOT NULL DEFAULT 0,
			is_email_verified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	return db
}

func createTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, nil, config.StorageConfig{})
	return service, db
}

func createTestUser(t *testing.T, db *gorm.DB, userID int64, email string) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO users (id, email, username, name, is_active, is_system_admin, is_email_verified) VALUES (?, ?, ?, ?, 1, 0, 1)`,
		userID, email, email, email,
	).Error
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	repo := infra.NewSupportTicketRepository(db)
	service := NewService(repo, nil, config.StorageConfig{})

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}
