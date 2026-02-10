package runner

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupPodEventHandlerDeps sets up dependencies for pod event handler testing
func setupPodEventHandlerDeps(t *testing.T) (*PodCoordinator, *RunnerConnectionManager, *TerminalRouter) {
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
	db := setupTestDB(t)

	// Create pods table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_key TEXT NOT NULL UNIQUE,
			runner_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			agent_status TEXT NOT NULL DEFAULT 'idle',
			pty_pid INTEGER,
			branch_name TEXT,
			sandbox_path TEXT,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			last_activity DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pods table: %v", err)
	}

	cm := NewRunnerConnectionManager(logger)
	tr := NewTerminalRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, db, logger)
	pc := NewPodCoordinator(db, cm, tr, hb, logger)

	return pc, cm, tr
}
