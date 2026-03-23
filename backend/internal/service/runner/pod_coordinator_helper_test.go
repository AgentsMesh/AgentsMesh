package runner

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// setupPodCoordinatorTestDB sets up database with pods table for testing.
// Returns gorm.DB (for raw SQL in tests) plus repository interfaces.
func setupPodCoordinatorTestDB(t *testing.T) (*gorm.DB, agentpod.PodRepository, runnerDomain.RunnerRepository) {
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
			agent_waiting_since DATETIME,
			finished_at DATETIME,
			alias TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pods table: %v", err)
	}

	podRepo := infra.NewPodRepository(db)
	runnerRepo := infra.NewRunnerRepository(db)
	return db, podRepo, runnerRepo
}

// setupPodCoordinatorDeps sets up dependencies for PodCoordinator testing
func setupPodCoordinatorDeps(t *testing.T) (*gorm.DB, *RunnerConnectionManager, *PodRouter, *HeartbeatBatcher, agentpod.PodRepository, runnerDomain.RunnerRepository) {
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
	db, podRepo, runnerRepo := setupPodCoordinatorTestDB(t)

	cm := NewRunnerConnectionManager(logger)
	tr := NewPodRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	return db, cm, tr, hb, podRepo, runnerRepo
}
