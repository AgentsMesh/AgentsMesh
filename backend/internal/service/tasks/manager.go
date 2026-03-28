package tasks

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
	"github.com/redis/go-redis/v9"
)

// StalePodCleaner marks initializing/running pods with stale activity as disconnected.
type StalePodCleaner interface {
	MarkStaleAsDisconnected(ctx context.Context, threshold time.Time) (int64, error)
}

// DeadLetterCleaner removes old dead letter entries past their TTL.
type DeadLetterCleaner interface {
	CleanupExpiredMessages(ctx context.Context, olderThan time.Time) (int64, error)
}

// DefaultConfig returns default task manager configuration
func DefaultConfig() Config {
	return Config{
		PipelinePollerInterval: 10 * time.Second,
		TaskProcessorInterval:  30 * time.Second,
		MRSyncInterval:         5 * time.Minute,
		PodCleanupInterval:     10 * time.Minute,
		DLQCleanupInterval:     24 * time.Hour,
		DLQRetentionTTL:        30 * 24 * time.Hour, // 30 days
		WorkerCount:            4,
		MaxQueueSize:           1000,
	}
}

// Config holds task manager configuration
type Config struct {
	PipelinePollerInterval time.Duration
	TaskProcessorInterval  time.Duration
	MRSyncInterval         time.Duration
	PodCleanupInterval     time.Duration
	DLQCleanupInterval     time.Duration
	DLQRetentionTTL        time.Duration
	WorkerCount            int
	MaxQueueSize           int
}

// Manager coordinates all background tasks
type Manager struct {
	podCleaner StalePodCleaner
	dlqCleaner DeadLetterCleaner
	redis      *redis.Client
	logger     *slog.Logger
	cfg        Config
	scheduler  *infraTasks.Scheduler
	workers    *infraTasks.WorkerPool
	wg         sync.WaitGroup

	// Services
	pipelinePoller *PipelinePollerService
	taskProcessor  *TaskProcessorService
}

// NewManager creates a new task manager
func NewManager(podCleaner StalePodCleaner, redisClient *redis.Client, logger *slog.Logger, cfg Config) *Manager {
	m := &Manager{
		podCleaner: podCleaner,
		redis:      redisClient,
		logger:     logger,
		cfg:        cfg,
	}

	// Initialize scheduler
	m.scheduler = infraTasks.NewScheduler(logger.With("component", "scheduler"))

	// Initialize worker pool
	m.workers = infraTasks.NewWorkerPool(
		logger.With("component", "workers"),
		infraTasks.WorkerPoolConfig{
			WorkerCount:  cfg.WorkerCount,
			MaxQueueSize: cfg.MaxQueueSize,
		},
	)

	// Initialize services
	m.pipelinePoller = NewPipelinePollerService(redisClient, logger.With("component", "pipeline_poller"))
	m.taskProcessor = NewTaskProcessorService(redisClient, logger.With("component", "task_processor"))

	return m
}

// RegisterTaskHandler registers a handler for processing completed tasks
func (m *Manager) RegisterTaskHandler(handler TaskHandler) {
	m.taskProcessor.RegisterHandler(handler)
}

// SetDeadLetterCleaner sets the dead-letter cleanup dependency.
// Must be called before Start if DLQ cleanup is desired.
func (m *Manager) SetDeadLetterCleaner(cleaner DeadLetterCleaner) {
	m.dlqCleaner = cleaner
}

// RegisterJobHandler registers a handler for background jobs
func (m *Manager) RegisterJobHandler(jobType string, handler infraTasks.JobHandler) {
	m.workers.RegisterHandler(jobType, handler)
}

// Start begins all background tasks
func (m *Manager) Start() error {
	m.logger.Info("starting task manager")

	// Register scheduled tasks
	m.registerScheduledTasks()

	// Start scheduler
	m.scheduler.Start()

	// Start worker pool
	m.workers.Start()

	// Monitor worker results
	m.wg.Add(1)
	go m.monitorWorkerResults()

	m.logger.Info("task manager started")
	return nil
}

// Stop gracefully stops all background tasks
func (m *Manager) Stop() {
	m.logger.Info("stopping task manager")
	m.scheduler.Stop()
	m.workers.Stop()
	m.wg.Wait()
	m.logger.Info("task manager stopped")
}

// registerScheduledTasks registers all scheduled tasks
func (m *Manager) registerScheduledTasks() {
	// Pipeline Poller - polls GitLab for pipeline status updates
	_ = m.scheduler.Register(&infraTasks.Task{
		Name:       "pipeline_poller",
		Interval:   m.cfg.PipelinePollerInterval,
		RunOnStart: true,
		Func: func(ctx context.Context) error {
			return m.pipelinePoller.Poll(ctx)
		},
	})

	// Task Processor - processes completed pipeline tasks
	_ = m.scheduler.Register(&infraTasks.Task{
		Name:       "task_processor",
		Interval:   m.cfg.TaskProcessorInterval,
		RunOnStart: false,
		Func: func(ctx context.Context) error {
			result, err := m.taskProcessor.Process(ctx)
			if err != nil {
				return err
			}
			if result.ProcessedCount > 0 {
				m.logger.Info("processed tasks",
					"count", result.ProcessedCount,
					"errors", len(result.Errors))
			}
			return nil
		},
	})

	// Pod Cleanup - cleans up stale pods
	_ = m.scheduler.Register(&infraTasks.Task{
		Name:       "pod_cleanup",
		Interval:   m.cfg.PodCleanupInterval,
		RunOnStart: false,
		Func: func(ctx context.Context) error {
			return m.cleanupStalePods(ctx)
		},
	})

	// DLQ Cleanup - removes dead letter entries older than TTL
	if m.dlqCleaner != nil {
		_ = m.scheduler.Register(&infraTasks.Task{
			Name:       "dlq_cleanup",
			Interval:   m.cfg.DLQCleanupInterval,
			RunOnStart: false,
			Func: func(ctx context.Context) error {
				return m.cleanupDeadLetters(ctx)
			},
		})
	}
}

// monitorWorkerResults monitors worker results for logging/metrics
func (m *Manager) monitorWorkerResults() {
	defer m.wg.Done()

	for result := range m.workers.Results() {
		if !result.Success {
			m.logger.Error("job failed",
				"job_id", result.JobID,
				"type", result.JobType,
				"error", result.Error,
				"duration", result.Duration,
				"retried", result.Retried)
		}
	}
}
