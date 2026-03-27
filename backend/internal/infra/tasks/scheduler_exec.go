package tasks

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// executeTask executes a single task with panic recovery
func (s *Scheduler) executeTask(task *Task) {
	start := time.Now()

	result := TaskResult{
		TaskName:  task.Name,
		StartTime: start,
	}

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			result.Error = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
			result.Success = false
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(start)

			s.logger.Error("task panicked",
				"task", task.Name,
				"error", result.Error,
				"duration", result.Duration)

			s.sendResult(result)
		}
	}()

	// Create task context with timeout (2x interval as safety margin)
	ctx, cancel := context.WithTimeout(s.ctx, task.Interval*2)
	defer cancel()

	err := task.Func(ctx)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(start)
	result.Error = err
	result.Success = err == nil

	if err != nil {
		s.logger.Error("task failed",
			"task", task.Name,
			"error", err,
			"duration", result.Duration)
	} else {
		s.logger.Debug("task completed",
			"task", task.Name,
			"duration", result.Duration)
	}

	s.sendResult(result)
}

// sendResult safely sends a result to the channel, checking stopped flag first.
func (s *Scheduler) sendResult(r TaskResult) {
	s.stoppedMu.RLock()
	stopped := s.stopped
	s.stoppedMu.RUnlock()

	if stopped {
		return
	}

	select {
	case s.results <- r:
	case <-s.ctx.Done():
	}
}

// processResults processes task results and notifies listeners
func (s *Scheduler) processResults() {
	defer s.wg.Done()

	for {
		select {
		case result, ok := <-s.results:
			if !ok {
				return
			}
			s.notifyListeners(result)
		case <-s.ctx.Done():
			s.drainResults()
			return
		}
	}
}

// drainResults processes any remaining buffered results after context cancellation.
func (s *Scheduler) drainResults() {
	for {
		select {
		case result, ok := <-s.results:
			if !ok {
				return
			}
			s.notifyListeners(result)
		default:
			return
		}
	}
}

// notifyListeners sends a result to all registered listeners.
func (s *Scheduler) notifyListeners(result TaskResult) {
	s.mu.RLock()
	listeners := s.listeners
	s.mu.RUnlock()

	for _, fn := range listeners {
		fn(result)
	}
}
