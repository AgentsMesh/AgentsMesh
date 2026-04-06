package ticket

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestService creates a Service backed by an in-memory DB for testing.
func newTestService(db *gorm.DB) *Service {
	return NewService(infra.NewTicketRepository(db))
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			slug TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			status TEXT NOT NULL DEFAULT 'backlog',
			priority TEXT NOT NULL DEFAULT 'none',
			severity TEXT,
			estimate INTEGER,
			due_date DATETIME,
			started_at DATETIME,
			completed_at DATETIME,
			repository_id INTEGER,
			reporter_id INTEGER NOT NULL,
			parent_ticket_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create tickets table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			repository_id INTEGER,
			name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '#808080',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create labels table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_labels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			label_id INTEGER NOT NULL
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_labels table: %v", err)
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

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_assignees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_assignees table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			parent_id INTEGER,
			mentions TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_comments table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ticket_merge_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			ticket_id INTEGER NOT NULL,
			pod_id INTEGER,
			mri_id INTEGER NOT NULL,
			mr_url TEXT NOT NULL UNIQUE,
			source_branch TEXT NOT NULL,
			target_branch TEXT NOT NULL DEFAULT 'main',
			title TEXT,
			state TEXT NOT NULL DEFAULT 'opened',
			pipeline_status TEXT,
			pipeline_id INTEGER,
			pipeline_url TEXT,
			merge_commit_sha TEXT,
			merged_at DATETIME,
			merged_by_id INTEGER,
			last_synced_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create ticket_merge_requests table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewServiceWithContext(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Verify service can be used with context
	_, _, err := service.ListTickets(ctx, &ticket.TicketListFilter{
		OrganizationID: 1,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("service should work with context: %v", err)
	}
}
