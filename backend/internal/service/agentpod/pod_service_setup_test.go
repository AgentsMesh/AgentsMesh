package agentpod

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	db.Exec(`CREATE TABLE IF NOT EXISTS runners (
		id INTEGER PRIMARY KEY,
		node_id TEXT,
		status TEXT,
		current_pods INTEGER DEFAULT 0
	)`)
	db.Exec("INSERT INTO runners (id, node_id, status, current_pods) VALUES (1, 'runner-001', 'online', 0)")

	db.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id INTEGER PRIMARY KEY,
		slug TEXT,
		title TEXT,
		content TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		username TEXT,
		name TEXT,
		email TEXT
	)`)
	db.Exec("INSERT INTO users (id, username, name, email) VALUES (1, 'testuser', 'Test User', 'test@example.com')")

	// GORM converts PtyPID -> pty_p_id, AgentPID -> agent_p_id
	// But service uses raw column names (pty_pid, agent_pid) in Updates()
	// We create columns for both to handle both cases
	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		pod_key TEXT NOT NULL UNIQUE,
		runner_id INTEGER NOT NULL,
		agent_slug INTEGER,
		custom_agent_slug INTEGER,
		repository_id INTEGER,
		ticket_id INTEGER,
		created_by_id INTEGER NOT NULL,
		pty_p_id INTEGER,
		pty_pid INTEGER,
		status TEXT NOT NULL DEFAULT 'initializing',
		agent_status TEXT NOT NULL DEFAULT 'idle',
		agent_p_id INTEGER,
		agent_pid INTEGER,
		started_at DATETIME,
		finished_at DATETIME,
		last_activity DATETIME,
		agent_waiting_since DATETIME,
		prompt TEXT,
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

	return db
}

// Helper functions
func intPtr(i int64) *int64 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// newTestPodService wraps *gorm.DB into PodRepository for testing.
func newTestPodService(db *gorm.DB) *PodService {
	return NewPodService(infra.NewPodRepository(db))
}

// newTestSettingsService wraps *gorm.DB into SettingsRepository for testing.
func newTestSettingsService(db *gorm.DB) *SettingsService {
	return NewSettingsService(infra.NewSettingsRepository(db))
}

// newTestAIProviderService wraps *gorm.DB into AIProviderRepository for testing.
// Accepts nil db for tests that don't hit the DB (pure logic tests).
func newTestAIProviderService(db *gorm.DB, enc *crypto.Encryptor) *AIProviderService {
	if db == nil {
		return NewAIProviderService(nil, enc)
	}
	return NewAIProviderService(infra.NewAIProviderRepository(db), enc)
}

// newTestAutopilotService wraps *gorm.DB into AutopilotRepository for testing.
func newTestAutopilotService(db *gorm.DB) *AutopilotControllerService {
	return NewAutopilotControllerService(infra.NewAutopilotRepository(db))
}

func TestNewPodService(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	if svc == nil {
		t.Error("NewPodService returned nil")
	}
	if svc.repo == nil {
		t.Error("Service repo not set correctly")
	}
}

// suppress unused import for agentpod domain
var _ = agentpod.StatusRunning
