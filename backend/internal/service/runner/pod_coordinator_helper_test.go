package runner

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// setupPodCoordinatorTestDB sets up database with pods table for testing
func setupPodCoordinatorTestDB(t *testing.T) *gorm.DB {
	db := setupTestDB(t)

	// Create tables for pods
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_key TEXT NOT NULL UNIQUE,
			runner_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			agent_status TEXT NOT NULL DEFAULT 'idle',
			error_code TEXT,
			error_message TEXT,
			last_activity DATETIME,
			finished_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pods table: %v", err)
	}

	return db
}

// setupPodCoordinatorDeps sets up dependencies for PodCoordinator testing
func setupPodCoordinatorDeps(t *testing.T) (*gorm.DB, *RunnerConnectionManager, *TerminalRouter, *HeartbeatBatcher) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() {
		mr.Close()
	})

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		redisClient.Close()
	})

	logger := newTestLogger()
	db := setupPodCoordinatorTestDB(t)

	cm := NewRunnerConnectionManager(logger)
	tr := NewTerminalRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, db, logger)

	return db, cm, tr, hb
}
