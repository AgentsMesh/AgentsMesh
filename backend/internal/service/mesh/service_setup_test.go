package mesh

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
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

	// Create tables
	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pod_key TEXT NOT NULL UNIQUE,
		organization_id INTEGER NOT NULL,
		ticket_id INTEGER,
		repository_id INTEGER,
		runner_id INTEGER,
		agent_slug INTEGER,
		custom_agent_slug INTEGER,
		created_by_id INTEGER NOT NULL,
		pty_pid INTEGER,
		status TEXT NOT NULL DEFAULT 'pending',
		agent_status TEXT NOT NULL DEFAULT 'idle',
		agent_pid INTEGER,
		started_at DATETIME,
		finished_at DATETIME,
		last_activity DATETIME,
		agent_waiting_since DATETIME,
		prompt TEXT NOT NULL DEFAULT '',
		branch_name TEXT,
		sandbox_path TEXT,
		model TEXT,
		permission_mode TEXT,
		think_level TEXT,
		error_code TEXT,
		error_message TEXT,
		title TEXT,
		alias TEXT,
		session_id TEXT,
		source_pod_key TEXT,
		credential_profile_id INTEGER,
		config_overrides TEXT DEFAULT '{}',
		interaction_mode TEXT NOT NULL DEFAULT 'pty',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT NOT NULL,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS channel_access (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		pod_key TEXT,
		user_id INTEGER,
		last_access DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func setupTestRepo(t *testing.T) (mesh.MeshRepository, *gorm.DB) {
	db := setupTestDB(t)
	repo := infra.NewMeshRepository(db)
	return repo, db
}

// ChannelPod for testing (local type to avoid import cycle)
type ChannelPod struct {
	ID        int64  `gorm:"primaryKey"`
	ChannelID int64  `gorm:"not null"`
	PodKey    string `gorm:"not null"`
}

func (ChannelPod) TableName() string {
	return "channel_pods"
}

// ChannelAccess for testing
type ChannelAccess struct {
	ID        int64   `gorm:"primaryKey"`
	ChannelID int64   `gorm:"not null"`
	PodKey    *string
	UserID    *int64
}

func (ChannelAccess) TableName() string {
	return "channel_access"
}

// Mock the channel.Message for count query
func init() {
	// Register table name mapping if needed
}
