package runner

import (
	"context"
	"time"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
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

// flushBatch writes a batch of heartbeats to the database via RunnerRepository.
// Each update is independent so one failure doesn't abort the entire batch.
func (b *HeartbeatBatcher) flushBatch(ctx context.Context, items []*HeartbeatItem) int {
	if len(items) == 0 {
		return 0
	}

	// Convert HeartbeatItem to domain HeartbeatUpdate
	domainItems := make([]runnerDomain.HeartbeatUpdate, len(items))
	for i, item := range items {
		domainItems[i] = runnerDomain.HeartbeatUpdate{
			RunnerID:    item.RunnerID,
			CurrentPods: item.CurrentPods,
			Status:      item.Status,
			Version:     item.Version,
			Timestamp:   item.Timestamp,
		}
	}

	updated, err := b.runnerRepo.BatchUpdateHeartbeats(ctx, domainItems)
	if err != nil {
		b.logger.Error("failed to batch update heartbeats", "error", err)
		return 0
	}
	return updated
}
