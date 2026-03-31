package tasks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWorkerPool(t *testing.T) *WorkerPool {
	t.Helper()
	return NewWorkerPool(newTestLogger(), WorkerPoolConfig{
		WorkerCount:  2,
		MaxQueueSize: 100,
	})
}

func TestWorkerPool_SubmitAndProcess(t *testing.T) {
	wp := newTestWorkerPool(t)

	var executed atomic.Bool
	wp.RegisterHandler("test-job", func(ctx context.Context, job *Job) error {
		executed.Store(true)
		return nil
	})

	wp.Start()

	err := wp.Submit(&Job{ID: "j1", Type: "test-job"})
	require.NoError(t, err)

	// Read result
	select {
	case r := <-wp.Results():
		assert.True(t, r.Success)
		assert.Equal(t, "j1", r.JobID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for job result")
	}

	wp.Stop()
	assert.True(t, executed.Load())
}

func TestWorkerPool_HandlerError(t *testing.T) {
	wp := newTestWorkerPool(t)

	wp.RegisterHandler("fail-job", func(ctx context.Context, job *Job) error {
		return fmt.Errorf("handler error")
	})

	wp.Start()

	err := wp.Submit(&Job{ID: "j-fail", Type: "fail-job", MaxRetry: 0})
	require.NoError(t, err)

	select {
	case r := <-wp.Results():
		assert.False(t, r.Success)
		assert.Contains(t, r.Error.Error(), "handler error")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for job result")
	}

	wp.Stop()
}

func TestWorkerPool_NoHandler(t *testing.T) {
	wp := newTestWorkerPool(t)
	wp.Start()

	err := wp.Submit(&Job{ID: "orphan", Type: "unknown-type"})
	require.NoError(t, err)

	select {
	case r := <-wp.Results():
		assert.False(t, r.Success)
		assert.Contains(t, r.Error.Error(), "no handler")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for job result")
	}

	wp.Stop()
}

func TestWorkerPool_Retry(t *testing.T) {
	wp := NewWorkerPool(
		slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})),
		WorkerPoolConfig{WorkerCount: 1, MaxQueueSize: 100},
	)

	var attempts atomic.Int32
	wp.RegisterHandler("retry-job", func(ctx context.Context, job *Job) error {
		n := attempts.Add(1)
		if n < 3 {
			return fmt.Errorf("transient error")
		}
		return nil
	})

	wp.Start()

	err := wp.Submit(&Job{
		ID:       "retry-1",
		Type:     "retry-job",
		MaxRetry: 3,
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	// Drain results until we see a successful one or timeout
	require.Eventually(t, func() bool {
		return attempts.Load() >= 3
	}, 10*time.Second, 50*time.Millisecond)

	wp.Stop()
}

func TestWorkerPool_GracefulStop(t *testing.T) {
	wp := newTestWorkerPool(t)
	wp.RegisterHandler("noop", func(ctx context.Context, job *Job) error {
		return nil
	})

	wp.Start()

	// Submit a few jobs
	for i := 0; i < 5; i++ {
		_ = wp.Submit(&Job{Type: "noop"})
	}

	done := make(chan struct{})
	go func() {
		wp.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not return in time")
	}
}

func TestWorkerPool_SubmitAfterStop(t *testing.T) {
	wp := newTestWorkerPool(t)
	wp.Start()
	wp.Stop()

	err := wp.Submit(&Job{Type: "late"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stopped")
}

func TestWorkerPool_QueueLength(t *testing.T) {
	// Use a pool with no workers started yet, so jobs stay in queue
	wp := NewWorkerPool(newTestLogger(), WorkerPoolConfig{
		WorkerCount:  0,
		MaxQueueSize: 10,
	})

	wp.RegisterHandler("wait", func(ctx context.Context, job *Job) error {
		return nil
	})

	// Don't start workers — submit directly to queue
	_ = wp.Submit(&Job{Type: "wait"})
	_ = wp.Submit(&Job{Type: "wait"})

	assert.Equal(t, 2, wp.QueueLength())

	// Clean up: cancel context so Stop does not block
	wp.cancel()
	wp.wg.Wait()
}

func TestWorkerPool_GetHandlerTypes(t *testing.T) {
	wp := newTestWorkerPool(t)

	wp.RegisterHandler("alpha", func(ctx context.Context, job *Job) error { return nil })
	wp.RegisterHandler("beta", func(ctx context.Context, job *Job) error { return nil })

	types := wp.GetHandlerTypes()
	assert.ElementsMatch(t, []string{"alpha", "beta"}, types)
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	wp := newTestWorkerPool(t)

	wp.RegisterHandler("panicky", func(ctx context.Context, job *Job) error {
		panic("oh no")
	})

	wp.Start()

	err := wp.Submit(&Job{ID: "panic-1", Type: "panicky"})
	require.NoError(t, err)

	select {
	case r := <-wp.Results():
		assert.False(t, r.Success)
		assert.Contains(t, r.Error.Error(), "panic")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for result")
	}

	wp.Stop()
}
