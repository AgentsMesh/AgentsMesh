package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentmesh/backend/internal/domain/runner"
	"github.com/redis/go-redis/v9"
)

// setupMiniredisForBatcher creates a miniredis instance for testing
func setupMiniredisForBatcher(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return mr, client
}

func TestNewHeartbeatBatcher(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)

	if batcher == nil {
		t.Fatal("NewHeartbeatBatcher returned nil")
	}
	if batcher.interval != DefaultFlushInterval {
		t.Errorf("interval: got %v, want %v", batcher.interval, DefaultFlushInterval)
	}
	if batcher.buffer == nil {
		t.Error("buffer should not be nil")
	}
}

func TestHeartbeatBatcherSetInterval(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
	batcher.SetInterval(1 * time.Second)

	if batcher.interval != 1*time.Second {
		t.Errorf("interval: got %v, want 1s", batcher.interval)
	}
}

func TestHeartbeatBatcherStartStop(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
	batcher.SetInterval(10 * time.Millisecond)

	// Start batcher
	batcher.Start()

	// Verify it's running
	batcher.mu.Lock()
	running := batcher.running
	batcher.mu.Unlock()
	if !running {
		t.Error("batcher should be running after Start")
	}

	// Start again should be no-op
	batcher.Start()

	// Stop batcher
	batcher.Stop()

	batcher.mu.Lock()
	running = batcher.running
	batcher.mu.Unlock()
	if running {
		t.Error("batcher should not be running after Stop")
	}

	// Stop again should be no-op
	batcher.Stop()
}

func TestHeartbeatBatcherRecordHeartbeat(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)

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
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)

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

func TestHeartbeatBatcherGetRunnerStatus(t *testing.T) {
	mr, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
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
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
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

func TestHeartbeatBatcherFlush(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	// Create a runner in the database
	runnerRecord := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		AuthTokenHash:  "hash",
		Status:         "offline",
	}
	if err := db.Create(runnerRecord).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	batcher := NewHeartbeatBatcher(redisClient, db, logger)

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
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)

	// Flush empty buffer should not panic
	batcher.Flush()

	if batcher.BufferSize() != 0 {
		t.Errorf("buffer size should be 0, got %d", batcher.BufferSize())
	}
}

func TestHeartbeatBatcherFlushLoop(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	// Create a runner in the database
	runnerRecord := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node-loop",
		AuthTokenHash:  "hash",
		Status:         "offline",
	}
	if err := db.Create(runnerRecord).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
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

func TestHeartbeatBatcherBufferSize(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
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

func TestHeartbeatBatcherFlushBatch(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	// Create multiple runners
	for i := 0; i < 5; i++ {
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "node-" + string(rune('A'+i)),
			AuthTokenHash:  "hash",
			Status:         "offline",
		}
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("failed to create runner: %v", err)
		}
	}

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
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

func TestHeartbeatBatcherConstants(t *testing.T) {
	// Verify constants are reasonable
	if DefaultFlushInterval != 5*time.Second {
		t.Errorf("DefaultFlushInterval: got %v, want 5s", DefaultFlushInterval)
	}
	if DefaultHeartbeatTTL != 90*time.Second {
		t.Errorf("DefaultHeartbeatTTL: got %v, want 90s", DefaultHeartbeatTTL)
	}
	if DefaultBatchSize != 100 {
		t.Errorf("DefaultBatchSize: got %d, want 100", DefaultBatchSize)
	}
	if HeartbeatOnlineThreshold != 90*time.Second {
		t.Errorf("HeartbeatOnlineThreshold: got %v, want 90s", HeartbeatOnlineThreshold)
	}
}

func TestHeartbeatBatcherRestartAfterStop(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, db, logger)
	batcher.SetInterval(10 * time.Millisecond)

	// Start, stop, restart
	batcher.Start()
	time.Sleep(20 * time.Millisecond)
	batcher.Stop()

	// Should be able to restart
	batcher.Start()
	batcher.mu.Lock()
	running := batcher.running
	batcher.mu.Unlock()
	if !running {
		t.Error("batcher should be running after restart")
	}

	batcher.Stop()
}
