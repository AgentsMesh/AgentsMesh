package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PipelinePollerService polls GitLab for pipeline status updates
type PipelinePollerService struct {
	db           *gorm.DB
	redis        *redis.Client
	watcher      *infraTasks.PipelineWatcher
	gitProviders map[int64]git.Provider
	logger       *slog.Logger
	mu           sync.RWMutex

	// Distributed lock
	lockKey     string
	lockTimeout time.Duration
}

// NewPipelinePollerService creates a new pipeline poller service
func NewPipelinePollerService(
	db *gorm.DB,
	redisClient *redis.Client,
	logger *slog.Logger,
) *PipelinePollerService {
	return &PipelinePollerService{
		db:           db,
		redis:        redisClient,
		watcher:      infraTasks.NewPipelineWatcher(redisClient, logger),
		gitProviders: make(map[int64]git.Provider),
		logger:       logger,
		lockKey:      infraTasks.PollerLockKey,
		lockTimeout:  time.Duration(infraTasks.LockTimeoutSeconds) * time.Second,
	}
}

// Poll executes one polling cycle
func (s *PipelinePollerService) Poll(ctx context.Context) error {
	// Try to acquire distributed lock
	acquired, err := s.acquireLock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		s.logger.Debug("another poller is running, skipping")
		return nil
	}
	defer s.releaseLock(ctx)

	// Update heartbeat
	s.updateHeartbeat(ctx)

	// Get watching list
	keys, err := s.watcher.GetWatchingKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get watching keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	s.logger.Debug("polling pipelines", "count", len(keys))

	var (
		updatedCount   int
		completedCount int
		skippedCount   int
		errors         []error
	)

	for _, key := range keys {
		result, err := s.pollSinglePipeline(ctx, key)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		switch result {
		case "updated":
			updatedCount++
		case "completed":
			completedCount++
		case "skipped":
			skippedCount++
		}
	}

	s.logger.Info("polling cycle completed",
		"watching", len(keys),
		"updated", updatedCount,
		"completed", completedCount,
		"skipped", skippedCount,
		"errors", len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during polling", len(errors))
	}

	return nil
}

// pollSinglePipeline polls a single pipeline
func (s *PipelinePollerService) pollSinglePipeline(ctx context.Context, key string) (string, error) {
	// Parse key format: projectID:pipelineID
	var projectID, pipelineID string
	if _, err := fmt.Sscanf(key, "%s:%s", &projectID, &pipelineID); err != nil {
		// Try alternative parsing
		parts := splitKey(key)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid key format: %s", key)
		}
		projectID = parts[0]
		pipelineID = parts[1]
	}

	// Skip job-level keys (updated via webhook)
	if len(pipelineID) > 4 && pipelineID[:4] == "job_" {
		return "skipped", nil
	}

	// Check if recently updated by webhook
	recent, err := s.watcher.IsRecentlyUpdated(ctx, projectID, pipelineID)
	if err != nil {
		s.logger.Warn("failed to check recent update", "key", key, "error", err)
	}
	if recent {
		return "skipped", nil
	}

	// Get current pipeline data
	pipeline, err := s.watcher.GetPipeline(ctx, projectID, pipelineID)
	if err != nil {
		return "", fmt.Errorf("failed to get pipeline %s: %w", key, err)
	}
	if pipeline == nil {
		return "skipped", nil
	}

	// Get Git provider for this project
	provider, err := s.getProvider(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get provider for %s: %w", projectID, err)
	}

	// Convert pipelineID to int
	pipelineIDInt, err := strconv.Atoi(pipelineID)
	if err != nil {
		return "", fmt.Errorf("invalid pipeline ID %s: %w", pipelineID, err)
	}

	// Query GitLab for current status
	pipelineInfo, err := provider.GetPipeline(ctx, projectID, pipelineIDInt)
	if err != nil {
		return "", fmt.Errorf("failed to get pipeline from GitLab: %w", err)
	}

	newStatus := pipelineInfo.Status
	webURL := pipelineInfo.WebURL

	// Update status in Redis
	if err := s.watcher.UpdateStatus(ctx, projectID, pipelineID, newStatus, webURL); err != nil {
		return "", fmt.Errorf("failed to update status: %w", err)
	}

	// Check if status changed
	if newStatus != pipeline.Status {
		s.logger.Info("pipeline status changed",
			"key", key,
			"old_status", pipeline.Status,
			"new_status", newStatus)
	}

	// Handle completed pipelines
	if infraTasks.TerminalStatuses[newStatus] {
		// Mark as processed
		if err := s.watcher.MarkProcessed(ctx, projectID, pipelineID); err != nil {
			s.logger.Warn("failed to mark pipeline as processed", "key", key, "error", err)
		}
		return "completed", nil
	}

	return "updated", nil
}

// getProvider retrieves or creates a Git provider for a project
func (s *PipelinePollerService) getProvider(ctx context.Context, projectID string) (git.Provider, error) {
	// For now, return a cached provider or create a new one
	// In production, this should look up the correct provider based on repository config
	s.mu.RLock()
	// Use 0 as a placeholder key - in real implementation, map projectID to provider
	if provider, ok := s.gitProviders[0]; ok {
		s.mu.RUnlock()
		return provider, nil
	}
	s.mu.RUnlock()

	// Provider not found - this is expected during initial setup
	// In production, look up provider from database
	return nil, fmt.Errorf("no provider configured for project %s", projectID)
}

// RegisterProvider registers a Git provider for use by the poller
func (s *PipelinePollerService) RegisterProvider(orgID int64, provider git.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gitProviders[orgID] = provider
}

// acquireLock tries to acquire the distributed lock
func (s *PipelinePollerService) acquireLock(ctx context.Context) (bool, error) {
	result, err := s.redis.SetNX(ctx, s.lockKey, "locked", s.lockTimeout).Result()
	return result, err
}

// releaseLock releases the distributed lock
func (s *PipelinePollerService) releaseLock(ctx context.Context) {
	s.redis.Del(ctx, s.lockKey)
}

// updateHeartbeat updates the poller heartbeat
func (s *PipelinePollerService) updateHeartbeat(ctx context.Context) {
	s.redis.SetEx(ctx, infraTasks.PollerHeartbeatKey, time.Now().UTC().Format(time.RFC3339), time.Duration(infraTasks.HeartbeatTTLSeconds)*time.Second)
}

// CheckHealth checks if the poller is healthy
func (s *PipelinePollerService) CheckHealth(ctx context.Context) (bool, error) {
	heartbeat, err := s.redis.Get(ctx, infraTasks.PollerHeartbeatKey).Result()
	if err == redis.Nil {
		// No heartbeat - check if there's anything to watch
		count, err := s.watcher.GetWatchingCount(ctx)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}
	if err != nil {
		return false, err
	}

	lastHeartbeat, err := time.Parse(time.RFC3339, heartbeat)
	if err != nil {
		return false, err
	}

	// Healthy if heartbeat is recent
	return time.Since(lastHeartbeat) < time.Duration(infraTasks.HeartbeatTTLSeconds*2)*time.Second, nil
}

// splitKey splits a pipeline key into parts
func splitKey(key string) []string {
	var parts []string
	current := ""
	for _, c := range key {
		if c == ':' && current != "" {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
