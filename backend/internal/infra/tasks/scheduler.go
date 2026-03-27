package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// TaskFunc is the function signature for a scheduled task
type TaskFunc func(ctx context.Context) error

// Task represents a scheduled task
type Task struct {
	Name     string
	Interval time.Duration
	Func     TaskFunc
	// RunOnStart determines if the task should run immediately when scheduled
	RunOnStart bool
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskName  string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Success   bool
}

// Scheduler manages scheduled background tasks
type Scheduler struct {
	tasks     map[string]*scheduledTask
	logger    *slog.Logger
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	results   chan TaskResult
	listeners []func(TaskResult)

	// stopped protects results channel from being written after close
	stopped   bool
	stoppedMu sync.RWMutex
	closeOnce sync.Once
}

// scheduledTask wraps a task with its control mechanisms
type scheduledTask struct {
	task   *Task
	ticker *time.Ticker
	stopCh chan struct{}
}

// NewScheduler creates a new task scheduler
func NewScheduler(logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		tasks:   make(map[string]*scheduledTask),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		results: make(chan TaskResult, 100),
	}
}

// Register adds a task to the scheduler
func (s *Scheduler) Register(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.Name]; exists {
		return fmt.Errorf("task %s already registered", task.Name)
	}

	st := &scheduledTask{
		task:   task,
		stopCh: make(chan struct{}),
	}
	s.tasks[task.Name] = st

	s.logger.Info("task registered",
		"task", task.Name,
		"interval", task.Interval)

	return nil
}

// Start begins executing all registered tasks
func (s *Scheduler) Start() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.wg.Add(1)
	go s.processResults()

	for name, st := range s.tasks {
		s.logger.Info("starting task", "task", name)
		st.ticker = time.NewTicker(st.task.Interval)

		s.wg.Add(1)
		go s.runTask(st)

		if st.task.RunOnStart {
			s.wg.Add(1)
			go func(task *Task) {
				defer s.wg.Done()
				s.executeTask(task)
			}(st.task)
		}
	}
}

// Stop gracefully stops all tasks
func (s *Scheduler) Stop() {
	s.logger.Info("stopping scheduler")

	s.stoppedMu.Lock()
	s.stopped = true
	s.stoppedMu.Unlock()

	s.cancel()

	s.mu.RLock()
	for _, st := range s.tasks {
		close(st.stopCh)
		if st.ticker != nil {
			st.ticker.Stop()
		}
	}
	s.mu.RUnlock()

	s.wg.Wait()
	s.logger.Info("all tasks stopped gracefully")

	s.closeOnce.Do(func() {
		close(s.results)
	})
}

// OnResult registers a callback for task results
func (s *Scheduler) OnResult(fn func(TaskResult)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, fn)
}

// runTask runs a scheduled task in a loop
func (s *Scheduler) runTask(st *scheduledTask) {
	defer s.wg.Done()

	for {
		select {
		case <-st.ticker.C:
			s.executeTask(st.task)
		case <-st.stopCh:
			return
		case <-s.ctx.Done():
			return
		}
	}
}

// RunNow executes a task immediately (outside of schedule)
func (s *Scheduler) RunNow(taskName string) error {
	s.mu.RLock()
	st, exists := s.tasks[taskName]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskName)
	}

	go s.executeTask(st.task)
	return nil
}

// GetTaskNames returns all registered task names
func (s *Scheduler) GetTaskNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tasks))
	for name := range s.tasks {
		names = append(names, name)
	}
	return names
}
