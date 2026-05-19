package tasks

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

func (s *Scheduler) executeTask(task *Task) {
	start := time.Now()

	result := TaskResult{
		TaskName:  task.Name,
		StartTime: start,
	}

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

func (s *Scheduler) notifyListeners(result TaskResult) {
	s.mu.RLock()
	listeners := s.listeners
	s.mu.RUnlock()

	for _, fn := range listeners {
		fn(result)
	}
}
