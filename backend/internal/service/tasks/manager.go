package tasks

import (
	"context"
	"log/slog"
	"sync"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
	"github.com/redis/go-redis/v9"
)

type StalePodCleaner interface {
	MarkStaleAsDisconnected(ctx context.Context, threshold time.Time) (int64, error)
}

type DeadLetterCleaner interface {
	CleanupExpiredMessages(ctx context.Context, olderThan time.Time) (int64, error)
}

func DefaultConfig() Config {
	return Config{
		PipelinePollerInterval: 10 * time.Second,
		TaskProcessorInterval:  30 * time.Second,
		MRSyncInterval:         5 * time.Minute,
		PodCleanupInterval:     10 * time.Minute,
		DLQCleanupInterval:     24 * time.Hour,
		DLQRetentionTTL:        30 * 24 * time.Hour,
		WorkerCount:            4,
		MaxQueueSize:           1000,
	}
}

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

type Manager struct {
	podCleaner StalePodCleaner
	dlqCleaner DeadLetterCleaner
	redis      *redis.Client
	logger     *slog.Logger
	cfg        Config
	scheduler  *infraTasks.Scheduler
	workers    *infraTasks.WorkerPool
	wg         sync.WaitGroup

	pipelinePoller *PipelinePollerService
	taskProcessor  *TaskProcessorService
}

func NewManager(podCleaner StalePodCleaner, redisClient *redis.Client, logger *slog.Logger, cfg Config) *Manager {
	m := &Manager{
		podCleaner: podCleaner,
		redis:      redisClient,
		logger:     logger,
		cfg:        cfg,
	}

	m.scheduler = infraTasks.NewScheduler(logger.With("component", "scheduler"))

	m.workers = infraTasks.NewWorkerPool(
		logger.With("component", "workers"),
		infraTasks.WorkerPoolConfig{
			WorkerCount:  cfg.WorkerCount,
			MaxQueueSize: cfg.MaxQueueSize,
		},
	)

	m.pipelinePoller = NewPipelinePollerService(redisClient, logger.With("component", "pipeline_poller"))
	m.taskProcessor = NewTaskProcessorService(redisClient, logger.With("component", "task_processor"))

	return m
}

func (m *Manager) RegisterTaskHandler(handler TaskHandler) {
	m.taskProcessor.RegisterHandler(handler)
}

// SetDeadLetterCleaner must be called before Start for DLQ cleanup to register.
func (m *Manager) SetDeadLetterCleaner(cleaner DeadLetterCleaner) {
	m.dlqCleaner = cleaner
}

func (m *Manager) RegisterJobHandler(jobType string, handler infraTasks.JobHandler) {
	m.workers.RegisterHandler(jobType, handler)
}

func (m *Manager) Start() error {
	m.logger.Info("starting task manager")

	m.registerScheduledTasks()

	m.scheduler.Start()

	m.workers.Start()

	m.wg.Add(1)
	go m.monitorWorkerResults()

	m.logger.Info("task manager started")
	return nil
}

func (m *Manager) Stop() {
	m.logger.Info("stopping task manager")
	m.scheduler.Stop()
	m.workers.Stop()
	m.wg.Wait()
	m.logger.Info("task manager stopped")
}

func (m *Manager) registerScheduledTasks() {
	_ = m.scheduler.Register(&infraTasks.Task{
		Name:       "pipeline_poller",
		Interval:   m.cfg.PipelinePollerInterval,
		RunOnStart: true,
		Func: func(ctx context.Context) error {
			return m.pipelinePoller.Poll(ctx)
		},
	})

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

	_ = m.scheduler.Register(&infraTasks.Task{
		Name:       "pod_cleanup",
		Interval:   m.cfg.PodCleanupInterval,
		RunOnStart: false,
		Func: func(ctx context.Context) error {
			return m.cleanupStalePods(ctx)
		},
	})

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
