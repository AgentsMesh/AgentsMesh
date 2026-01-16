package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	// DefaultFlushInterval is the default interval for flushing heartbeats to database
	DefaultFlushInterval = 5 * time.Second

	// DefaultHeartbeatTTL is the Redis TTL for heartbeat status (3x heartbeat interval for safety)
	DefaultHeartbeatTTL = 90 * time.Second

	// DefaultBatchSize is the number of heartbeats to process in a single batch
	DefaultBatchSize = 100

	// HeartbeatOnlineThreshold is the threshold to consider a runner online (same as TTL)
	HeartbeatOnlineThreshold = 90 * time.Second
)

// HeartbeatBatcher aggregates heartbeat updates and flushes them to the database in batches
// This reduces database write pressure from 3,333/sec (100K runners / 30s) to ~667/sec (5s batch)
type HeartbeatBatcher struct {
	redis    *redis.Client
	db       *gorm.DB
	logger   *slog.Logger

	buffer   map[int64]*HeartbeatItem
	mu       sync.Mutex
	interval time.Duration

	// Lifecycle management - protected by mu
	stopCh  chan struct{}
	doneCh  chan struct{}
	running bool
}

// HeartbeatItem represents a pending heartbeat update
type HeartbeatItem struct {
	RunnerID    int64
	CurrentPods int
	Status      string
	Version     string
	Timestamp   time.Time
}

// RedisRunnerStatus represents runner status stored in Redis for real-time queries
type RedisRunnerStatus struct {
	LastHeartbeat int64  `json:"last_heartbeat"`
	CurrentPods   int    `json:"current_pods"`
	Status        string `json:"status"`
	Version       string `json:"version,omitempty"`
}

// NewHeartbeatBatcher creates a new heartbeat batcher
func NewHeartbeatBatcher(redisClient *redis.Client, db *gorm.DB, logger *slog.Logger) *HeartbeatBatcher {
	return &HeartbeatBatcher{
		redis:    redisClient,
		db:       db,
		logger:   logger,
		buffer:   make(map[int64]*HeartbeatItem),
		interval: DefaultFlushInterval,
		// stopCh and doneCh are created in Start() to allow restart
	}
}

// SetInterval sets the flush interval (for testing)
func (b *HeartbeatBatcher) SetInterval(interval time.Duration) {
	b.interval = interval
}

// Start starts the background flush loop
func (b *HeartbeatBatcher) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	// Create new channels for this lifecycle (allows restart after Stop)
	b.stopCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	stopCh := b.stopCh
	doneCh := b.doneCh
	b.mu.Unlock()

	go b.flushLoop(stopCh, doneCh)
	b.logger.Info("heartbeat batcher started", "interval", b.interval)
}

// Stop stops the batcher and flushes remaining items
func (b *HeartbeatBatcher) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	stopCh := b.stopCh
	doneCh := b.doneCh
	b.mu.Unlock()

	close(stopCh)
	<-doneCh
	b.logger.Info("heartbeat batcher stopped")
}

// RecordHeartbeat records a heartbeat for batch processing
// It immediately updates Redis for real-time status queries, and buffers DB writes
func (b *HeartbeatBatcher) RecordHeartbeat(ctx context.Context, runnerID int64, currentPods int, status, version string) error {
	now := time.Now()

	// 1. Immediately update Redis for real-time status queries
	// This allows SelectAvailableRunner to get fresh data without DB round-trip
	redisKey := fmt.Sprintf("runner:%d:status", runnerID)
	statusData := map[string]interface{}{
		"last_heartbeat": now.Unix(),
		"current_pods":   currentPods,
		"status":         status,
	}
	if version != "" {
		statusData["version"] = version
	}

	pipe := b.redis.Pipeline()
	pipe.HSet(ctx, redisKey, statusData)
	pipe.Expire(ctx, redisKey, DefaultHeartbeatTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		b.logger.Warn("failed to update runner status in Redis",
			"runner_id", runnerID,
			"error", err)
		// Continue anyway - DB write will still happen
	}

	// 2. Buffer for batch DB write
	b.mu.Lock()
	b.buffer[runnerID] = &HeartbeatItem{
		RunnerID:    runnerID,
		CurrentPods: currentPods,
		Status:      status,
		Version:     version,
		Timestamp:   now,
	}
	b.mu.Unlock()

	return nil
}

// GetRunnerStatus gets runner status from Redis cache (for real-time queries)
// Returns nil if not found in cache
func (b *HeartbeatBatcher) GetRunnerStatus(ctx context.Context, runnerID int64) (*RedisRunnerStatus, error) {
	redisKey := fmt.Sprintf("runner:%d:status", runnerID)
	result, err := b.redis.HGetAll(ctx, redisKey).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil // Not found
	}

	status := &RedisRunnerStatus{
		Status: result["status"],
	}

	if ts, ok := result["last_heartbeat"]; ok {
		if v, err := strconv.ParseInt(ts, 10, 64); err == nil {
			status.LastHeartbeat = v
		}
	}
	if pods, ok := result["current_pods"]; ok {
		if v, err := strconv.Atoi(pods); err == nil {
			status.CurrentPods = v
		}
	}
	if version, ok := result["version"]; ok {
		status.Version = version
	}

	return status, nil
}

// IsRunnerOnline checks if a runner is online based on Redis cache
func (b *HeartbeatBatcher) IsRunnerOnline(ctx context.Context, runnerID int64) bool {
	status, err := b.GetRunnerStatus(ctx, runnerID)
	if err != nil || status == nil {
		return false
	}

	// Check if last heartbeat is within timeout threshold
	return time.Now().Unix()-status.LastHeartbeat < int64(HeartbeatOnlineThreshold.Seconds())
}

// flushLoop runs the periodic flush
func (b *HeartbeatBatcher) flushLoop(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-stopCh:
			// Final flush before exit
			b.flush()
			return
		}
	}
}

// flush writes all buffered heartbeats to the database
func (b *HeartbeatBatcher) flush() {
	// Swap buffer atomically
	b.mu.Lock()
	if len(b.buffer) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buffer
	b.buffer = make(map[int64]*HeartbeatItem)
	b.mu.Unlock()

	// Process in batches for better performance
	items := make([]*HeartbeatItem, 0, len(batch))
	for _, item := range batch {
		items = append(items, item)
	}

	ctx := context.Background()
	start := time.Now()
	totalUpdated := 0

	for i := 0; i < len(items); i += DefaultBatchSize {
		end := i + DefaultBatchSize
		if end > len(items) {
			end = len(items)
		}
		batchItems := items[i:end]

		updated := b.flushBatch(ctx, batchItems)
		totalUpdated += updated
	}

	b.logger.Debug("flushed heartbeat batch",
		"total", len(batch),
		"updated", totalUpdated,
		"duration", time.Since(start))
}

// flushBatch writes a batch of heartbeats to the database using bulk update
func (b *HeartbeatBatcher) flushBatch(ctx context.Context, items []*HeartbeatItem) int {
	if len(items) == 0 {
		return 0
	}

	// Use transaction for batch update
	tx := b.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		b.logger.Error("failed to begin transaction", "error", tx.Error)
		return 0
	}

	updated := 0
	for _, item := range items {
		updates := map[string]interface{}{
			"last_heartbeat": item.Timestamp,
			"current_pods":   item.CurrentPods,
			"status":         item.Status,
		}
		if item.Version != "" {
			updates["runner_version"] = item.Version
		}

		result := tx.Model(&runner.Runner{}).
			Where("id = ?", item.RunnerID).
			Updates(updates)

		if result.Error != nil {
			b.logger.Error("failed to update runner heartbeat",
				"runner_id", item.RunnerID,
				"error", result.Error)
			continue
		}
		if result.RowsAffected > 0 {
			updated++
		}
	}

	if err := tx.Commit().Error; err != nil {
		b.logger.Error("failed to commit heartbeat batch", "error", err)
		tx.Rollback()
		return 0
	}

	return updated
}

// BufferSize returns the current buffer size (for monitoring)
func (b *HeartbeatBatcher) BufferSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}

// Flush immediately flushes all buffered heartbeats to the database
// This is useful for testing and graceful shutdown scenarios
func (b *HeartbeatBatcher) Flush() {
	b.flush()
}
