package runner

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

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
