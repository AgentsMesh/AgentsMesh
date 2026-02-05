package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetPipeline retrieves pipeline data from Redis
func (pw *PipelineWatcher) GetPipeline(ctx context.Context, projectID, pipelineID string) (*WatchedPipeline, error) {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	data, err := pw.redis.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline data: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	pipeline := &WatchedPipeline{
		ProjectID:    data["project_id"],
		PipelineID:   data["pipeline_id"],
		TaskType:     data["task_type"],
		Status:       data["status"],
		WebURL:       data["web_url"],
		ArtifactPath: data["artifact_path"],
		JobName:      data["job_name"],
		ResultJSON:   data["result_json"],
	}

	if taskID, err := strconv.ParseInt(data["task_id"], 10, 64); err == nil {
		pipeline.TaskID = taskID
	}

	if updatedAt, err := time.Parse(time.RFC3339, data["updated_at"]); err == nil {
		pipeline.UpdatedAt = updatedAt
	}

	if metadataStr := data["metadata"]; metadataStr != "" {
		_ = json.Unmarshal([]byte(metadataStr), &pipeline.Metadata)
	}

	return pipeline, nil
}

// GetCompletedPipelines retrieves all completed pipelines of a specific type
func (pw *PipelineWatcher) GetCompletedPipelines(ctx context.Context, taskType string) ([]*WatchedPipeline, error) {
	// Get all watching keys
	keys, err := pw.redis.SMembers(ctx, WatchingSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get watching set: %w", err)
	}

	var completed []*WatchedPipeline

	for _, key := range keys {
		hashKey := PipelineKeyPrefix + key
		data, err := pw.redis.HGetAll(ctx, hashKey).Result()
		if err != nil {
			pw.logger.Warn("failed to get pipeline data", "key", key, "error", err)
			continue
		}

		// Filter by task type
		if data["task_type"] != taskType {
			continue
		}

		// Check if status is terminal
		status := data["status"]
		if !TerminalStatuses[status] {
			continue
		}

		pipeline := &WatchedPipeline{
			ProjectID:    data["project_id"],
			PipelineID:   data["pipeline_id"],
			TaskType:     data["task_type"],
			Status:       status,
			WebURL:       data["web_url"],
			ArtifactPath: data["artifact_path"],
			JobName:      data["job_name"],
			ResultJSON:   data["result_json"],
		}

		if taskID, err := strconv.ParseInt(data["task_id"], 10, 64); err == nil {
			pipeline.TaskID = taskID
		}

		if updatedAt, err := time.Parse(time.RFC3339, data["updated_at"]); err == nil {
			pipeline.UpdatedAt = updatedAt
		}

		completed = append(completed, pipeline)
	}

	return completed, nil
}

// GetWatchingCount returns the number of pipelines being watched
func (pw *PipelineWatcher) GetWatchingCount(ctx context.Context) (int64, error) {
	return pw.redis.SCard(ctx, WatchingSetKey).Result()
}

// GetWatchingKeys returns all keys being watched
func (pw *PipelineWatcher) GetWatchingKeys(ctx context.Context) ([]string, error) {
	return pw.redis.SMembers(ctx, WatchingSetKey).Result()
}

// IsRecentlyUpdated checks if a pipeline was updated recently (by webhook)
func (pw *PipelineWatcher) IsRecentlyUpdated(ctx context.Context, projectID, pipelineID string) (bool, error) {
	key := fmt.Sprintf("%s:%s", projectID, pipelineID)
	hashKey := PipelineKeyPrefix + key

	updatedAtStr, err := pw.redis.HGet(ctx, hashKey, "updated_at").Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return false, nil
	}

	return time.Since(updatedAt) < RecentUpdateThreshold, nil
}
