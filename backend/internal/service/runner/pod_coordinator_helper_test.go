package runner

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func setupPodCoordinatorTestDB(t *testing.T) (*gorm.DB, PodStore, runnerDomain.RunnerRepository) {
	db := setupTestDB(t)

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
	podStore := agentpodSvc.NewPodService(podRepo)
	runnerRepo := infra.NewRunnerRepository(db)
	return db, podStore, runnerRepo
}

func setupPodCoordinatorDeps(t *testing.T) (*gorm.DB, *RunnerConnectionManager, *PodRouter, *HeartbeatBatcher, PodStore, runnerDomain.RunnerRepository) {
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
	db, podStore, runnerRepo := setupPodCoordinatorTestDB(t)

	cm := NewRunnerConnectionManager(logger)
	tr := NewPodRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	return db, cm, tr, hb, podStore, runnerRepo
}
