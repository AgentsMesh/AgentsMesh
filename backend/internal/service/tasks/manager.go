package tasks

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// DefaultConfig returns default task manager configuration
func DefaultConfig() Config {
	return Config{
		PipelinePollerInterval: 10 * time.Second,
		TaskProcessorInterval:  30 * time.Second,
		MRSyncInterval:         5 * time.Minute,
		PodCleanupInterval:     10 * time.Minute,
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
	WorkerCount            int
	MaxQueueSize           int
}

// Manager coordinates all background tasks
type Manager struct {
	db        *gorm.DB
	redis     *redis.Client
	logger    *slog.Logger
	cfg       Config
	scheduler *infraTasks.Scheduler
	workers   *infraTasks.WorkerPool
	wg        sync.WaitGroup

	// Services
	pipelinePoller *PipelinePollerService
	taskProcessor  *TaskProcessorService
}

// NewManager creates a new task manager
func NewManager(db *gorm.DB, redisClient *redis.Client, logger *slog.Logger, cfg Config) *Manager {
	m := &Manager{
		db:     db,
		redis:  redisClient,
		logger: logger,
		cfg:    cfg,
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
	m.pipelinePoller = NewPipelinePollerService(db, redisClient, logger.With("component", "pipeline_poller"))
	m.taskProcessor = NewTaskProcessorService(db, redisClient, logger.With("component", "task_processor"))

	return m
}

// RegisterTaskHandler registers a handler for processing completed tasks
func (m *Manager) RegisterTaskHandler(handler TaskHandler) {
	m.taskProcessor.RegisterHandler(handler)
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

// cleanupStalePods cleans up pods that are no longer active
func (m *Manager) cleanupStalePods(ctx context.Context) error {
	// Update pods with stale heartbeats
	staleThreshold := time.Now().Add(-30 * time.Minute)

	result := m.db.WithContext(ctx).
		Table("pods").
		Where("status IN ?", []string{"initializing", "running"}).
		Where("last_activity < ? OR last_activity IS NULL", staleThreshold).
		Update("status", "disconnected")

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		m.logger.Info("cleaned up stale pods",
			"count", result.RowsAffected)
	}

	return nil
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

// Health represents the health status of the task manager
type Health struct {
	Healthy             bool   `json:"healthy"`
	PollerHealthy       bool   `json:"poller_healthy"`
	WatchingCount       int64  `json:"watching_count"`
	QueueLength         int    `json:"queue_length"`
	ScheduledTasks      int    `json:"scheduled_tasks"`
	RegisteredHandlers  int    `json:"registered_handlers"`
}

// CheckHealth returns the health status of the task manager
func (m *Manager) CheckHealth(ctx context.Context) (*Health, error) {
	health := &Health{
		QueueLength:        m.workers.QueueLength(),
		ScheduledTasks:     len(m.scheduler.GetTaskNames()),
		RegisteredHandlers: len(m.workers.GetHandlerTypes()),
	}

	// Check poller health
	pollerHealthy, err := m.pipelinePoller.CheckHealth(ctx)
	if err != nil {
		m.logger.Warn("failed to check poller health", "error", err)
	}
	health.PollerHealthy = pollerHealthy

	// Get watching count
	watcher := infraTasks.NewPipelineWatcher(m.redis, m.logger)
	watchingCount, err := watcher.GetWatchingCount(ctx)
	if err != nil {
		m.logger.Warn("failed to get watching count", "error", err)
	}
	health.WatchingCount = watchingCount

	// Overall health
	health.Healthy = health.PollerHealthy && health.QueueLength < m.cfg.MaxQueueSize

	return health, nil
}
