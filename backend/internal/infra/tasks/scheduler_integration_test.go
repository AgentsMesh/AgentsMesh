package tasks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestScheduler_RegisterAndRun(t *testing.T) {
	s := NewScheduler(newTestLogger())

	var count atomic.Int32
	err := s.Register(&Task{
		Name:     "counter",
		Interval: 10 * time.Millisecond,
		Func: func(ctx context.Context) error {
			count.Add(1)
			return nil
		},
	})
	require.NoError(t, err)

	s.Start()

	// Wait for at least one tick
	require.Eventually(t, func() bool {
		return count.Load() >= 1
	}, time.Second, 5*time.Millisecond)

	s.Stop()
	assert.GreaterOrEqual(t, count.Load(), int32(1))
}

func TestScheduler_RunOnStart(t *testing.T) {
	s := NewScheduler(newTestLogger())

	var called atomic.Bool
	err := s.Register(&Task{
		Name:       "immediate",
		Interval:   10 * time.Second, // long interval so only RunOnStart triggers
		RunOnStart: true,
		Func: func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
	})
	require.NoError(t, err)

	s.Start()

	require.Eventually(t, func() bool {
		return called.Load()
	}, time.Second, 5*time.Millisecond)

	s.Stop()
}

func TestScheduler_RunNow(t *testing.T) {
	s := NewScheduler(newTestLogger())

	var called atomic.Bool
	err := s.Register(&Task{
		Name:     "manual",
		Interval: 10 * time.Second, // long interval; only RunNow should fire
		Func: func(ctx context.Context) error {
			called.Store(true)
			return nil
		},
	})
	require.NoError(t, err)

	// Need to Start the scheduler so processResults goroutine is running.
	s.Start()

	err = s.RunNow("manual")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return called.Load()
	}, time.Second, 5*time.Millisecond)

	s.Stop()
}

func TestScheduler_RunNow_NotFound(t *testing.T) {
	s := NewScheduler(newTestLogger())

	err := s.RunNow("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestScheduler_GracefulStop(t *testing.T) {
	s := NewScheduler(newTestLogger())

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("task-%d", i)
		err := s.Register(&Task{
			Name:     name,
			Interval: 10 * time.Millisecond,
			Func: func(ctx context.Context) error {
				return nil
			},
		})
		require.NoError(t, err)
	}

	s.Start()
	time.Sleep(50 * time.Millisecond) // let tasks fire a few times

	// Stop should return without hanging
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return in time — possible goroutine leak")
	}
}

func TestScheduler_DuplicateRegistration(t *testing.T) {
	s := NewScheduler(newTestLogger())

	task := &Task{
		Name:     "dup",
		Interval: time.Second,
		Func:     func(ctx context.Context) error { return nil },
	}

	err := s.Register(task)
	require.NoError(t, err)

	err = s.Register(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestScheduler_GetTaskNames(t *testing.T) {
	s := NewScheduler(newTestLogger())

	for _, name := range []string{"a", "b", "c"} {
		err := s.Register(&Task{
			Name:     name,
			Interval: time.Second,
			Func:     func(ctx context.Context) error { return nil },
		})
		require.NoError(t, err)
	}

	names := s.GetTaskNames()
	assert.Len(t, names, 3)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, names)
}

func TestScheduler_OnResult(t *testing.T) {
	s := NewScheduler(newTestLogger())

	var mu sync.Mutex
	var results []TaskResult

	s.OnResult(func(r TaskResult) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	err := s.Register(&Task{
		Name:     "observed",
		Interval: 10 * time.Millisecond,
		Func:     func(ctx context.Context) error { return nil },
	})
	require.NoError(t, err)

	s.Start()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(results) >= 1
	}, time.Second, 5*time.Millisecond)

	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(results), 1)
	assert.Equal(t, "observed", results[0].TaskName)
	assert.True(t, results[0].Success)
}

func TestScheduler_TaskError(t *testing.T) {
	s := NewScheduler(newTestLogger())

	var mu sync.Mutex
	var results []TaskResult

	s.OnResult(func(r TaskResult) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	})

	errExpected := fmt.Errorf("boom")
	err := s.Register(&Task{
		Name:     "failing",
		Interval: 10 * time.Millisecond,
		Func:     func(ctx context.Context) error { return errExpected },
	})
	require.NoError(t, err)

	s.Start()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(results) >= 1
	}, time.Second, 5*time.Millisecond)

	s.Stop()

	mu.Lock()
	defer mu.Unlock()
	assert.False(t, results[0].Success)
	assert.NotNil(t, results[0].Error)
}
