package runner

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// setupPodEventHandlerDeps sets up dependencies for pod event handler testing
func setupPodEventHandlerDeps(t *testing.T) (*PodCoordinator, *RunnerConnectionManager, *PodRouter, *gorm.DB) {
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

	podRepo := infra.NewPodRepository(db)
	podStore := agentpodSvc.NewPodService(podRepo)
	runnerRepo := infra.NewRunnerRepository(db)

	cm := NewRunnerConnectionManager(logger)
	tr := NewPodRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)

	return pc, cm, tr, db
}
