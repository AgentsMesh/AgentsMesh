package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type JobPriority int

const (
	PriorityLow JobPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

type Job struct {
	ID       string
	Type     string
	Payload  map[string]interface{}
	Priority JobPriority
	MaxRetry int
	Timeout  time.Duration

	retryCount int
	createdAt  time.Time
	startedAt  time.Time
}

type JobHandler func(ctx context.Context, job *Job) error

type JobResult struct {
	JobID     string
	JobType   string
	Success   bool
	Error     error
	Duration  time.Duration
	Retried   bool
}

type WorkerPool struct {
	handlers map[string]JobHandler
	jobs     chan *Job
	results  chan JobResult
	logger   *slog.Logger
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc

	workerCount int
	maxQueueSize int
}

type WorkerPoolConfig struct {
	WorkerCount  int
	MaxQueueSize int
}

func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		WorkerCount:  4,
		MaxQueueSize: 1000,
	}
}

func NewWorkerPool(logger *slog.Logger, cfg WorkerPoolConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		handlers:     make(map[string]JobHandler),
		jobs:         make(chan *Job, cfg.MaxQueueSize),
		results:      make(chan JobResult, cfg.MaxQueueSize),
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		workerCount:  cfg.WorkerCount,
		maxQueueSize: cfg.MaxQueueSize,
	}
}

func (wp *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.handlers[jobType] = handler
	wp.logger.Info("job handler registered", "type", jobType)
}

func (wp *WorkerPool) Start() {
	wp.logger.Info("starting worker pool",
		"workers", wp.workerCount,
		"queue_size", wp.maxQueueSize)

	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

func (wp *WorkerPool) Stop() {
	wp.logger.Info("stopping worker pool")
	wp.cancel()

	wp.wg.Wait()
	close(wp.jobs)
	close(wp.results)

	wp.logger.Info("worker pool stopped gracefully")
}

func (wp *WorkerPool) Submit(job *Job) error {
	select {
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is stopped")
	default:
	}

	if job.ID == "" {
		job.ID = fmt.Sprintf("%s-%d", job.Type, time.Now().UnixNano())
	}
	job.createdAt = time.Now()

	if job.Timeout == 0 {
		job.Timeout = 5 * time.Minute
	}

	select {
	case wp.jobs <- job:
		wp.logger.Debug("job submitted",
			"job_id", job.ID,
			"type", job.Type,
			"priority", job.Priority)
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is stopped")
	default:
		return fmt.Errorf("job queue is full")
	}
}

func (wp *WorkerPool) Results() <-chan JobResult {
	return wp.results
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("worker started", "worker_id", id)

	for {
		select {
		case <-wp.ctx.Done():
			return
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}

			result := wp.processJob(job)

			select {
			case wp.results <- result:
			case <-wp.ctx.Done():
				return
			}
		}
	}
}

func (wp *WorkerPool) processJob(job *Job) JobResult {
	job.startedAt = time.Now()
	result := JobResult{
		JobID:   job.ID,
		JobType: job.Type,
	}

	wp.mu.RLock()
	handler, exists := wp.handlers[job.Type]
	wp.mu.RUnlock()

	if !exists {
		result.Error = fmt.Errorf("no handler for job type: %s", job.Type)
		result.Success = false
		result.Duration = time.Since(job.startedAt)
		return result
	}

	ctx, cancel := context.WithTimeout(wp.ctx, job.Timeout)
	defer cancel()

	func() {
		defer func() {
			if r := recover(); r != nil {
				result.Error = fmt.Errorf("panic in job handler: %v", r)
			}
		}()
		result.Error = handler(ctx, job)
	}()

	result.Duration = time.Since(job.startedAt)
	result.Success = result.Error == nil

	if result.Error != nil && job.retryCount < job.MaxRetry {
		job.retryCount++
		result.Retried = true

		wp.logger.Warn("job failed, retrying",
			"job_id", job.ID,
			"type", job.Type,
			"retry", job.retryCount,
			"max_retry", job.MaxRetry,
			"error", result.Error)

		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			backoff := time.Duration(job.retryCount) * time.Second
			select {
			case <-time.After(backoff):
				_ = wp.Submit(job)
			case <-wp.ctx.Done():
				return
			}
		}()
	} else if result.Error != nil {
		wp.logger.Error("job failed permanently",
			"job_id", job.ID,
			"type", job.Type,
			"error", result.Error,
			"duration", result.Duration)
	} else {
		wp.logger.Debug("job completed",
			"job_id", job.ID,
			"type", job.Type,
			"duration", result.Duration)
	}

	return result
}

func (wp *WorkerPool) QueueLength() int {
	return len(wp.jobs)
}

func (wp *WorkerPool) GetHandlerTypes() []string {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	types := make([]string, 0, len(wp.handlers))
	for t := range wp.handlers {
		types = append(types, t)
	}
	return types
}
