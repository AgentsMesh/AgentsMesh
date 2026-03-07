package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestHeartbeatBatcherGetRunnerStatus(t *testing.T) {
	mr, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	ctx := context.Background()

	// Test not found
	status, err := batcher.GetRunnerStatus(ctx, 999)
	if err != nil {
		t.Fatalf("GetRunnerStatus error: %v", err)
	}
	if status != nil {
		t.Error("status should be nil for non-existent runner")
	}

	// Set up test data in Redis
	now := time.Now().Unix()
	mr.HSet("runner:123:status", "last_heartbeat", fmt.Sprintf("%d", now))
	mr.HSet("runner:123:status", "current_pods", "3")
	mr.HSet("runner:123:status", "status", "online")
	mr.HSet("runner:123:status", "version", "2.0.0")

	// Get status
	status, err = batcher.GetRunnerStatus(ctx, 123)
	if err != nil {
		t.Fatalf("GetRunnerStatus error: %v", err)
	}
	if status == nil {
		t.Fatal("status should not be nil")
	}
	if status.LastHeartbeat != now {
		t.Errorf("LastHeartbeat: got %d, want %d", status.LastHeartbeat, now)
	}
	if status.CurrentPods != 3 {
		t.Errorf("CurrentPods: got %d, want 3", status.CurrentPods)
	}
	if status.Status != "online" {
		t.Errorf("Status: got %q, want %q", status.Status, "online")
	}
	if status.Version != "2.0.0" {
		t.Errorf("Version: got %q, want %q", status.Version, "2.0.0")
	}
}

func TestHeartbeatBatcherIsRunnerOnline(t *testing.T) {
	mr, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	ctx := context.Background()

	// Test non-existent runner
	if batcher.IsRunnerOnline(ctx, 999) {
		t.Error("non-existent runner should not be online")
	}

	// Test recent heartbeat
	now := time.Now().Unix()
	mr.HSet("runner:100:status", "last_heartbeat", fmt.Sprintf("%d", now))
	if !batcher.IsRunnerOnline(ctx, 100) {
		t.Error("runner with recent heartbeat should be online")
	}

	// Test old heartbeat (beyond threshold)
	oldTime := time.Now().Add(-HeartbeatOnlineThreshold - time.Minute).Unix()
	mr.HSet("runner:101:status", "last_heartbeat", fmt.Sprintf("%d", oldTime))
	if batcher.IsRunnerOnline(ctx, 101) {
		t.Error("runner with old heartbeat should not be online")
	}
}
