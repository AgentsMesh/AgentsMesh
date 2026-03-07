package tasks

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// mockPodCleaner implements StalePodCleaner for testing.
type mockPodCleaner struct {
	markStaleResult int64
	markStaleErr    error
	markStaleCalls  int
}

func (m *mockPodCleaner) MarkStaleAsDisconnected(_ context.Context, _ time.Time) (int64, error) {
	m.markStaleCalls++
	return m.markStaleResult, m.markStaleErr
}

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"PipelinePollerInterval", cfg.PipelinePollerInterval, 10 * time.Second},
		{"TaskProcessorInterval", cfg.TaskProcessorInterval, 30 * time.Second},
		{"MRSyncInterval", cfg.MRSyncInterval, 5 * time.Minute},
		{"PodCleanupInterval", cfg.PodCleanupInterval, 10 * time.Minute},
		{"WorkerCount", cfg.WorkerCount, 4},
		{"MaxQueueSize", cfg.MaxQueueSize, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestHealthStruct(t *testing.T) {
	health := &Health{
		Healthy:            true,
		PollerHealthy:      true,
		WatchingCount:      10,
		QueueLength:        5,
		ScheduledTasks:     3,
		RegisteredHandlers: 2,
	}

	if !health.Healthy {
		t.Error("expected healthy to be true")
	}
	if health.WatchingCount != 10 {
		t.Errorf("WatchingCount = %d, want 10", health.WatchingCount)
	}
}

func TestConfigValues(t *testing.T) {
	cfg := Config{
		PipelinePollerInterval: 5 * time.Second,
		TaskProcessorInterval:  15 * time.Second,
		MRSyncInterval:         1 * time.Minute,
		PodCleanupInterval:     5 * time.Minute,
		WorkerCount:            8,
		MaxQueueSize:           500,
	}

	if cfg.WorkerCount != 8 {
		t.Errorf("WorkerCount = %d, want 8", cfg.WorkerCount)
	}
	if cfg.MaxQueueSize != 500 {
		t.Errorf("MaxQueueSize = %d, want 500", cfg.MaxQueueSize)
	}
}

func TestNewManager(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.podCleaner != podCleaner {
		t.Error("expected manager.podCleaner to be set")
	}
	if manager.redis != redisClient {
		t.Error("expected manager.redis to be set")
	}
	if manager.pipelinePoller == nil {
		t.Error("expected pipelinePoller to be initialized")
	}
	if manager.taskProcessor == nil {
		t.Error("expected taskProcessor to be initialized")
	}
}

func TestManager_StartStop(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := Config{
		PipelinePollerInterval: 1 * time.Hour, // Long interval to avoid actual polling
		TaskProcessorInterval:  1 * time.Hour,
		MRSyncInterval:         1 * time.Hour,
		PodCleanupInterval:     1 * time.Hour,
		WorkerCount:            2,
		MaxQueueSize:           100,
	}

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	// Start manager
	err := manager.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)

	// Stop manager
	manager.Stop()
}

func TestManager_GetScheduledTasks(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)
	manager.Start()
	defer manager.Stop()

	tasks := manager.GetScheduledTasks()
	if len(tasks) == 0 {
		t.Error("expected scheduled tasks to be registered")
	}

	// Should have pipeline_poller, task_processor, pod_cleanup
	expectedTasks := map[string]bool{
		"pipeline_poller": false,
		"task_processor":  false,
		"pod_cleanup":     false,
	}

	for _, task := range tasks {
		if _, ok := expectedTasks[task]; ok {
			expectedTasks[task] = true
		}
	}

	for name, found := range expectedTasks {
		if !found {
			t.Errorf("expected task %s to be registered", name)
		}
	}
}

func TestManager_GetQueueLength(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	length := manager.GetQueueLength()
	if length != 0 {
		t.Errorf("expected queue length 0, got %d", length)
	}
}

func TestManager_GetJobHandlerTypes(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	types := manager.GetJobHandlerTypes()
	// Initially empty
	if types == nil {
		t.Error("expected non-nil handler types slice")
	}
}

func TestManager_CleanupStalePods(t *testing.T) {
	podCleaner := &mockPodCleaner{markStaleResult: 1}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	err := manager.cleanupStalePods(context.Background())
	if err != nil {
		t.Fatalf("cleanupStalePods() error = %v", err)
	}

	if podCleaner.markStaleCalls != 1 {
		t.Errorf("expected 1 call to MarkStaleAsDisconnected, got %d", podCleaner.markStaleCalls)
	}
}

func TestManager_CleanupStalePods_NoStale(t *testing.T) {
	podCleaner := &mockPodCleaner{markStaleResult: 0}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	err := manager.cleanupStalePods(context.Background())
	if err != nil {
		t.Fatalf("cleanupStalePods() error = %v", err)
	}

	if podCleaner.markStaleCalls != 1 {
		t.Errorf("expected 1 call to MarkStaleAsDisconnected, got %d", podCleaner.markStaleCalls)
	}
}

func TestManager_CleanupStalePods_Error(t *testing.T) {
	podCleaner := &mockPodCleaner{markStaleErr: errors.New("db error")}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	err := manager.cleanupStalePods(context.Background())
	if err == nil {
		t.Fatal("expected error from cleanupStalePods")
	}
}

func TestManager_GetPipelineWatcher(t *testing.T) {
	podCleaner := &mockPodCleaner{}
	_, redisClient := setupTestRedis(t)
	logger := testLogger()
	cfg := DefaultConfig()

	manager := NewManager(podCleaner, redisClient, logger, cfg)

	watcher := manager.GetPipelineWatcher()
	if watcher == nil {
		t.Error("expected non-nil pipeline watcher")
	}
}
