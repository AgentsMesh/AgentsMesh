package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

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

// flushBatch writes a batch of heartbeats to the database using independent updates.
// Each update is independent so one failure doesn't abort the entire batch
// (PostgreSQL marks a transaction as aborted after any SQL error).
func (b *HeartbeatBatcher) flushBatch(ctx context.Context, items []*HeartbeatItem) int {
	if len(items) == 0 {
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

		result := b.db.WithContext(ctx).
			Model(&runner.Runner{}).
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

	return updated
}
