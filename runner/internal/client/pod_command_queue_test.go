package client

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPodCommandQueue_SequentialPerPod(t *testing.T) {
	q := NewPodCommandQueue()
	defer q.Wait()

	var order []string
	var mu sync.Mutex
	done := make(chan struct{})

	// Enqueue two commands for the same pod — must execute in order.
	q.Enqueue("pod-1", func() {
		time.Sleep(50 * time.Millisecond) // simulate slow create_pod
		mu.Lock()
		order = append(order, "create_pod")
		mu.Unlock()
	})
	q.Enqueue("pod-1", func() {
		mu.Lock()
		order = append(order, "create_autopilot")
		mu.Unlock()
		close(done)
	})

	<-done
	q.Remove("pod-1")

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 || order[0] != "create_pod" || order[1] != "create_autopilot" {
		t.Errorf("Expected [create_pod, create_autopilot], got %v", order)
	}
}

func TestPodCommandQueue_LongChainOrder(t *testing.T) {
	q := NewPodCommandQueue()

	var order []int
	var mu sync.Mutex
	const n = 10

	for i := 0; i < n; i++ {
		i := i
		q.Enqueue("pod-1", func() {
			mu.Lock()
			order = append(order, i)
			mu.Unlock()
		})
	}

	q.Remove("pod-1")
	q.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(order) != n {
		t.Fatalf("Expected %d commands, got %d", n, len(order))
	}
	for i := 0; i < n; i++ {
		if order[i] != i {
			t.Errorf("Command %d executed at position %d", i, order[i])
		}
	}
}

func TestPodCommandQueue_ConcurrentAcrossPods(t *testing.T) {
	q := NewPodCommandQueue()
	defer q.Wait()

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		podKey := fmt.Sprintf("pod-%d", i)
		wg.Add(1)
		q.Enqueue(podKey, func() {
			defer wg.Done()
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
		})
	}

	wg.Wait()
	for i := 0; i < 5; i++ {
		q.Remove(fmt.Sprintf("pod-%d", i))
	}

	if maxConcurrent.Load() < 2 {
		t.Errorf("Expected concurrent execution across pods, max was %d", maxConcurrent.Load())
	}
}

func TestPodCommandQueue_RemoveDrainsRemaining(t *testing.T) {
	q := NewPodCommandQueue()

	var count atomic.Int32

	q.Enqueue("pod-1", func() { count.Add(1) })
	q.Enqueue("pod-1", func() { count.Add(1) })
	q.Enqueue("pod-1", func() { count.Add(1) })

	q.Remove("pod-1")
	q.Wait()

	if count.Load() != 3 {
		t.Errorf("Expected 3 commands executed, got %d", count.Load())
	}
}

func TestPodCommandQueue_DoubleRemoveNoPanic(t *testing.T) {
	q := NewPodCommandQueue()

	q.Enqueue("pod-1", func() {})
	q.Remove("pod-1")
	q.Remove("pod-1") // should not panic
	q.Wait()
}

func TestPodCommandQueue_EnqueueAfterRemove(t *testing.T) {
	// Simulates session recovery: same pod_key reused after previous instance was removed.
	q := NewPodCommandQueue()

	var count atomic.Int32

	// First lifecycle
	q.Enqueue("pod-1", func() { count.Add(1) })
	q.Remove("pod-1")
	q.Wait()

	if count.Load() != 1 {
		t.Fatalf("First lifecycle: expected 1, got %d", count.Load())
	}

	// Second lifecycle — new worker should be created for the same pod_key
	q.Enqueue("pod-1", func() { count.Add(1) })
	q.Remove("pod-1")
	q.Wait()

	if count.Load() != 2 {
		t.Errorf("Second lifecycle: expected 2, got %d", count.Load())
	}
}

func TestPodCommandQueue_PanicRecovery(t *testing.T) {
	// A panic in one command must not kill the worker or block subsequent commands.
	q := NewPodCommandQueue()

	var executed atomic.Int32
	done := make(chan struct{})

	q.Enqueue("pod-1", func() {
		executed.Add(1)
		panic("simulated crash")
	})
	q.Enqueue("pod-1", func() {
		executed.Add(1)
		close(done)
	})

	select {
	case <-done:
		// second command executed despite first panicking
	case <-time.After(3 * time.Second):
		t.Fatal("Second command was not executed after panic in first")
	}

	q.Remove("pod-1")
	q.Wait()

	if executed.Load() != 2 {
		t.Errorf("Expected 2 commands executed, got %d", executed.Load())
	}
}

func TestPodCommandQueue_EmptyWaitNonBlocking(t *testing.T) {
	q := NewPodCommandQueue()
	// Wait on a queue with no pods should return immediately
	done := make(chan struct{})
	go func() {
		q.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait() blocked on empty queue")
	}
}

func TestPodCommandQueue_ConcurrentCreateAndRemove(t *testing.T) {
	// Race detector stress test: many pods created and removed concurrently.
	q := NewPodCommandQueue()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		podKey := fmt.Sprintf("pod-%d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.Enqueue(podKey, func() {
				time.Sleep(time.Millisecond)
			})
			q.Enqueue(podKey, func() {})
			q.Remove(podKey)
		}()
	}
	wg.Wait()
	q.Wait()
}
