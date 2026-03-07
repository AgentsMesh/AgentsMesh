package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestNewHeartbeatBatcher(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)

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
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
	batcher.SetInterval(1 * time.Second)

	if batcher.interval != 1*time.Second {
		t.Errorf("interval: got %v, want 1s", batcher.interval)
	}
}

func TestHeartbeatBatcherStartStop(t *testing.T) {
	_, redisClient := setupMiniredisForBatcher(t)
	db := setupTestDB(t)
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
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
	runnerRepo := infra.NewRunnerRepository(db)
	logger := newTestLogger()

	batcher := NewHeartbeatBatcher(redisClient, runnerRepo, logger)
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
