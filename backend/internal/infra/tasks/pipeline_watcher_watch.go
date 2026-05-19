package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (pw *PipelineWatcher) Watch(ctx context.Context, projectID, pipelineID string, taskType string, taskID int64, metadata map[string]interface{}) error {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	if err := pw.redis.SAdd(ctx, WatchingSetKey, key).Err(); err != nil {
		return fmt.Errorf("failed to add to watching set: %w", err)
	}

	data := map[string]interface{}{
		"project_id":  projectID,
		"pipeline_id": pipelineID,
		"task_type":   taskType,
		"task_id":     taskID,
		"status":      "pending",
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	}

	if metadata != nil {
		if artifactPath, ok := metadata["artifact_path"].(string); ok {
			data["artifact_path"] = artifactPath
		}
		if jobName, ok := metadata["job_name"].(string); ok {
			data["job_name"] = jobName
		}
		if metaJSON, err := json.Marshal(metadata); err == nil {
			data["metadata"] = string(metaJSON)
		}
	}

	if err := pw.redis.HSet(ctx, hashKey, data).Err(); err != nil {
		return fmt.Errorf("failed to store pipeline data: %w", err)
	}

	pw.logger.Info("pipeline watch started",
		"project_id", projectID,
		"pipeline_id", pipelineID,
		"task_type", taskType,
		"task_id", taskID)

	return nil
}

func (pw *PipelineWatcher) UpdateStatus(ctx context.Context, projectID, pipelineID, status string, webURL string) error {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	data := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if webURL != "" {
		data["web_url"] = webURL
	}

	if err := pw.redis.HSet(ctx, hashKey, data).Err(); err != nil {
		return fmt.Errorf("failed to update pipeline status: %w", err)
	}

	if TerminalStatuses[status] {
		pw.redis.SRem(ctx, WatchingSetKey, key)
		pw.redis.Expire(ctx, hashKey, CompletedPipelineTTL)
	}

	return nil
}

func (pw *PipelineWatcher) Unwatch(ctx context.Context, projectID, pipelineID string) error {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	pw.redis.SRem(ctx, WatchingSetKey, key)

	pw.redis.Del(ctx, hashKey)

	return nil
}
