package infra

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupLoopTestDB creates an in-memory SQLite database for testing.
// Creates loop-related tables plus minimal pods/autopilot_controllers tables for SSOT queries.
func setupLoopTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create loops table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			agent_slug INTEGER,
			custom_agent_slug INTEGER,
			permission_mode TEXT NOT NULL DEFAULT 'bypassPermissions',
			prompt_template TEXT NOT NULL DEFAULT '',
			repository_id INTEGER,
			runner_id INTEGER,
			branch_name TEXT,
			ticket_id INTEGER,
			credential_profile_id INTEGER,
			config_overrides BLOB DEFAULT NULL,
			prompt_variables BLOB DEFAULT NULL,
			execution_mode TEXT NOT NULL DEFAULT 'autopilot',
			cron_expression TEXT,
			autopilot_config BLOB DEFAULT NULL,
			callback_url TEXT,
			status TEXT NOT NULL DEFAULT 'enabled',
			sandbox_strategy TEXT NOT NULL DEFAULT 'persistent',
			session_persistence INTEGER NOT NULL DEFAULT 1,
			concurrency_policy TEXT NOT NULL DEFAULT 'skip',
			max_concurrent_runs INTEGER NOT NULL DEFAULT 1,
			max_retained_runs INTEGER NOT NULL DEFAULT 0,
			timeout_minutes INTEGER NOT NULL DEFAULT 60,
			idle_timeout_sec INTEGER NOT NULL DEFAULT 30,
			sandbox_path TEXT,
			last_pod_key TEXT,
			created_by_id INTEGER NOT NULL DEFAULT 0,
			total_runs INTEGER NOT NULL DEFAULT 0,
			successful_runs INTEGER NOT NULL DEFAULT 0,
			failed_runs INTEGER NOT NULL DEFAULT 0,
			last_run_at DATETIME,
			next_run_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, slug)
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create loops table: %v", err)
	}

	// Create loop_runs table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS loop_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			loop_id INTEGER NOT NULL,
			run_number INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			pod_key TEXT,
			autopilot_controller_key TEXT,
			trigger_type TEXT NOT NULL DEFAULT 'manual',
			trigger_source TEXT,
			trigger_params BLOB DEFAULT NULL,
			resolved_prompt TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			duration_sec INTEGER,
			exit_summary TEXT,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create loop_runs table: %v", err)
	}

	// Create minimal pods table for SSOT queries
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'initializing',
			agent_status TEXT NOT NULL DEFAULT 'idle',
			agent_waiting_since DATETIME,
			finished_at DATETIME,
			alias TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pods table: %v", err)
	}

	// Create minimal autopilot_controllers table for SSOT queries
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS autopilot_controllers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			autopilot_controller_key TEXT NOT NULL UNIQUE,
			phase TEXT NOT NULL DEFAULT 'initializing',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create autopilot_controllers table: %v", err)
	}

	return db
}

// Helper functions for creating test data
func loopStrPtr(s string) *string { return &s }
