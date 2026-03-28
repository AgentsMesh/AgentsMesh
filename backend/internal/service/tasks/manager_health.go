package tasks

import (
	"context"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
)

// Health represents the health status of the task manager
type Health struct {
	Healthy            bool  `json:"healthy"`
	PollerHealthy      bool  `json:"poller_healthy"`
	WatchingCount      int64 `json:"watching_count"`
	QueueLength        int   `json:"queue_length"`
	ScheduledTasks     int   `json:"scheduled_tasks"`
	RegisteredHandlers int   `json:"registered_handlers"`
}

// CheckHealth returns the health status of the task manager
func (m *Manager) CheckHealth(ctx context.Context) (*Health, error) {
	health := &Health{
		QueueLength:        m.workers.QueueLength(),
		ScheduledTasks:     len(m.scheduler.GetTaskNames()),
		RegisteredHandlers: len(m.workers.GetHandlerTypes()),
	}

	pollerHealthy, err := m.pipelinePoller.CheckHealth(ctx)
	if err != nil {
		m.logger.Warn("failed to check poller health", "error", err)
	}
	health.PollerHealthy = pollerHealthy

	watcher := infraTasks.NewPipelineWatcher(m.redis, m.logger)
	watchingCount, err := watcher.GetWatchingCount(ctx)
	if err != nil {
		m.logger.Warn("failed to get watching count", "error", err)
	}
	health.WatchingCount = watchingCount

	health.Healthy = health.PollerHealthy && health.QueueLength < m.cfg.MaxQueueSize
	return health, nil
}

// SubmitJob submits a job to the worker pool
func (m *Manager) SubmitJob(job *infraTasks.Job) error {
	return m.workers.Submit(job)
}

// RunTaskNow triggers a scheduled task to run immediately
func (m *Manager) RunTaskNow(taskName string) error {
	return m.scheduler.RunNow(taskName)
}

// GetScheduledTasks returns all scheduled task names
func (m *Manager) GetScheduledTasks() []string {
	return m.scheduler.GetTaskNames()
}

// GetJobHandlerTypes returns all registered job handler types
func (m *Manager) GetJobHandlerTypes() []string {
	return m.workers.GetHandlerTypes()
}

// GetQueueLength returns the current job queue length
func (m *Manager) GetQueueLength() int {
	return m.workers.QueueLength()
}

// GetPipelineWatcher returns the pipeline watcher for webhook handlers
func (m *Manager) GetPipelineWatcher() *infraTasks.PipelineWatcher {
	return infraTasks.NewPipelineWatcher(m.redis, m.logger)
}

// cleanupStalePods cleans up pods that are no longer active
func (m *Manager) cleanupStalePods(ctx context.Context) error {
	staleThreshold := time.Now().Add(-30 * time.Minute)

	rowsAffected, err := m.podCleaner.MarkStaleAsDisconnected(ctx, staleThreshold)
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		m.logger.Info("cleaned up stale pods",
			"count", rowsAffected)
	}

	return nil
}

// cleanupDeadLetters removes dead letter entries older than the configured TTL.
func (m *Manager) cleanupDeadLetters(ctx context.Context) error {
	olderThan := time.Now().Add(-m.cfg.DLQRetentionTTL)

	rowsAffected, err := m.dlqCleaner.CleanupExpiredMessages(ctx, olderThan)
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		m.logger.Info("cleaned up dead letter entries",
			"count", rowsAffected,
			"retention", m.cfg.DLQRetentionTTL)
	}

	return nil
}
