package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestHeartbeatBatcherFlush(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	// Create a runner in the database
	runnerRecord := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         "offline",
	}
	if err := db.Create(runnerRecord).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	ctx := context.Background()

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerRecord.ID, 5, "online", "1.0.0")
	if err != nil {
		t.Fatalf("RecordHeartbeat error: %v", err)
	}

	// Manually flush
	batcher.Flush()

	// Verify buffer is empty
	if batcher.BufferSize() != 0 {
		t.Errorf("buffer should be empty after flush, got %d", batcher.BufferSize())
	}

	// Verify database was updated
	var updatedRunner runner.Runner
	if err := db.First(&updatedRunner, runnerRecord.ID).Error; err != nil {
		t.Fatalf("failed to get runner: %v", err)
	}
	if updatedRunner.Status != "online" {
		t.Errorf("runner status: got %q, want %q", updatedRunner.Status, "online")
	}
	if updatedRunner.CurrentPods != 5 {
		t.Errorf("runner current_pods: got %d, want 5", updatedRunner.CurrentPods)
	}
}

func TestHeartbeatBatcherFlushEmptyBuffer(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

	// Flush empty buffer should not panic
	batcher.Flush()

	if batcher.BufferSize() != 0 {
		t.Errorf("buffer size should be 0, got %d", batcher.BufferSize())
	}
}

func TestHeartbeatBatcherFlushLoop(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	// Create a runner in the database
	runnerRecord := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node-loop",
		Status:         "offline",
	}
	if err := db.Create(runnerRecord).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	batcher.SetInterval(50 * time.Millisecond)

	ctx := context.Background()

	// Record heartbeat
	err := batcher.RecordHeartbeat(ctx, runnerRecord.ID, 3, "online", "")
	if err != nil {
		t.Fatalf("RecordHeartbeat error: %v", err)
	}

	// Start batcher
	batcher.Start()

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	// Stop batcher
	batcher.Stop()

	// Verify database was updated
	var updatedRunner runner.Runner
	if err := db.First(&updatedRunner, runnerRecord.ID).Error; err != nil {
		t.Fatalf("failed to get runner: %v", err)
	}
	if updatedRunner.Status != "online" {
		t.Errorf("runner status: got %q, want %q", updatedRunner.Status, "online")
	}
}

func TestHeartbeatBatcherFlushBatch(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	// Create multiple runners
	for i := 0; i < 5; i++ {
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "node-" + string(rune('A'+i)),
			Status:         "offline",
		}
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("failed to create runner: %v", err)
		}
	}

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	ctx := context.Background()

	// Record heartbeats for all runners
	for i := int64(1); i <= 5; i++ {
		batcher.RecordHeartbeat(ctx, i, int(i), "online", "")
	}

	// Flush
	batcher.Flush()

	// Verify all runners were updated
	var runners []runner.Runner
	if err := db.Find(&runners).Error; err != nil {
		t.Fatalf("failed to get runners: %v", err)
	}

	for _, r := range runners {
		if r.Status != "online" {
			t.Errorf("runner %d status: got %q, want %q", r.ID, r.Status, "online")
		}
	}
}
