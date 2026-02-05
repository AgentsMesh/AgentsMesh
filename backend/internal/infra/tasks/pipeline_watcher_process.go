package tasks

import (
	"context"
	"fmt"
)

// MarkProcessed marks a pipeline as processed (removes from watching)
func (pw *PipelineWatcher) MarkProcessed(ctx context.Context, projectID, pipelineID string) error {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	// Remove from watching set
	if err := pw.redis.SRem(ctx, WatchingSetKey, key).Err(); err != nil {
		return fmt.Errorf("failed to remove from watching set: %w", err)
	}

	// Update processed flag
	if err := pw.redis.HSet(ctx, hashKey, "processed", "true").Err(); err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	// Set TTL for cleanup
	pw.redis.Expire(ctx, hashKey, CompletedPipelineTTL)

	return nil
}

// StoreArtifact stores artifact data for a pipeline
func (pw *PipelineWatcher) StoreArtifact(ctx context.Context, projectID, pipelineID string, resultJSON string) error {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	return pw.redis.HSet(ctx, hashKey, "result_json", resultJSON).Err()
}
