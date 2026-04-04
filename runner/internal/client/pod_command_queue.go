package client

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// PodCommandQueue dispatches commands per pod sequentially.
// Each pod_key gets a dedicated buffered channel and worker goroutine.
// Different pods run concurrently; same pod's commands execute in order.
//
// This eliminates race conditions between dependent commands (e.g., create_pod
// must complete before create_autopilot can find the pod in the store).
type PodCommandQueue struct {
	queues sync.Map // map[string]chan func()
	wg     sync.WaitGroup
}

// NewPodCommandQueue creates a new per-pod command queue.
func NewPodCommandQueue() *PodCommandQueue {
	return &PodCommandQueue{}
}

// podQueueSize is the buffer size for per-pod command channels.
// 16 is generous — a pod rarely has more than a few commands in flight.
const podQueueSize = 16

// Enqueue sends a command to the pod's queue for sequential execution.
// The worker goroutine is lazily created on first enqueue for each pod.
// This method never blocks (buffered channel).
func (q *PodCommandQueue) Enqueue(podKey string, fn func()) {
	ch := q.getOrCreate(podKey)
	ch <- fn
}

// getOrCreate returns the channel for a pod, creating a worker if needed.
func (q *PodCommandQueue) getOrCreate(podKey string) chan func() {
	if v, ok := q.queues.Load(podKey); ok {
		return v.(chan func())
	}
	ch := make(chan func(), podQueueSize)
	if actual, loaded := q.queues.LoadOrStore(podKey, ch); loaded {
		// Another goroutine created it first — use theirs.
		return actual.(chan func())
	}
	// We won the race — start worker for this pod.
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		for fn := range ch {
			q.safeExec(fn)
		}
	}()
	return ch
}

// safeExec runs fn with panic recovery so one bad command doesn't kill the worker.
func (q *PodCommandQueue) safeExec(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			logger.GRPC().Error("Panic in pod command", "panic", r)
		}
	}()
	fn()
}

// Remove closes and removes a pod's queue after the pod is fully terminated.
// Any pending commands in the channel will be drained by the worker before it exits.
func (q *PodCommandQueue) Remove(podKey string) {
	if v, ok := q.queues.LoadAndDelete(podKey); ok {
		close(v.(chan func()))
	}
}

// Wait waits for all pod workers to finish (used during shutdown).
func (q *PodCommandQueue) Wait() {
	q.wg.Wait()
}
