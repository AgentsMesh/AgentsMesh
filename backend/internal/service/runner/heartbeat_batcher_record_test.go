package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestHeartbeatBatcherRecordHeartbeat(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	ctx := context.Background()
	runnerID := int64(123)

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerID, 5, "online", "1.0.0")
	if err != nil {
		t.Fatalf("RecordHeartbeat error: %v", err)
	}

	// Verify Redis was updated
	key := "runner:123:status"
	result, err := redisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("HGetAll error: %v", err)
	}
	if result["status"] != "online" {
		t.Errorf("redis status: got %q, want %q", result["status"], "online")
	}
	if result["current_pods"] != "5" {
		t.Errorf("redis current_pods: got %q, want %q", result["current_pods"], "5")
	}
	if result["version"] != "1.0.0" {
		t.Errorf("redis version: got %q, want %q", result["version"], "1.0.0")
	}

	// Verify buffer was updated
	if batcher.BufferSize() != 1 {
		t.Errorf("buffer size: got %d, want 1", batcher.BufferSize())
	}
}

func TestHeartbeatBatcherRecordHeartbeatWithoutVersion(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	ctx := context.Background()
	runnerID := int64(456)

	// Record heartbeat without version
	err := batcher.RecordHeartbeat(ctx, runnerID, 2, "online", "")
	if err != nil {
		t.Fatalf("RecordHeartbeat error: %v", err)
	}

	// Verify version is not set
	key := "runner:456:status"
	result, _ := redisClient.HGetAll(ctx, key).Result()
	if _, exists := result["version"]; exists {
		t.Error("version should not be set when empty")
	}
}

func TestHeartbeatBatcherBufferSize(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	ctx := context.Background()

	// Initially empty
	if batcher.BufferSize() != 0 {
		t.Errorf("initial buffer size: got %d, want 0", batcher.BufferSize())
	}

	// Record multiple heartbeats
	for i := int64(1); i <= 5; i++ {
		batcher.RecordHeartbeat(ctx, i, int(i), "online", "")
	}

	if batcher.BufferSize() != 5 {
		t.Errorf("buffer size after 5 records: got %d, want 5", batcher.BufferSize())
	}

	// Same runner updates should not increase buffer size
	batcher.RecordHeartbeat(ctx, 1, 10, "online", "")
	if batcher.BufferSize() != 5 {
		t.Errorf("buffer size after update: got %d, want 5", batcher.BufferSize())
	}
}
