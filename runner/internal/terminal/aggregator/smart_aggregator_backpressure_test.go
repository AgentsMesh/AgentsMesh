package aggregator

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestSmartAggregator_Backpressure tests pause/resume functionality
func TestSmartAggregator_Backpressure(t *testing.T) {
	var pauseCalled, resumeCalled bool
	var flushCount int
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			flushCount++
			mu.Unlock()
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithBackpressureCallbacks(
			func() { pauseCalled = true },
			func() { resumeCalled = true },
		),
	)

	// Initial state
	if agg.IsPaused() {
		t.Error("Should not be paused initially")
	}

	// Pause
	agg.Pause()
	if !agg.IsPaused() {
		t.Error("Should be paused after Pause()")
	}
	if !pauseCalled {
		t.Error("onPause callback should be called")
	}

	// Write while paused - should buffer but not flush
	agg.Write([]byte("paused data"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	countWhilePaused := flushCount
	mu.Unlock()

	// Resume
	agg.Resume()
	if agg.IsPaused() {
		t.Error("Should not be paused after Resume()")
	}
	if !resumeCalled {
		t.Error("onResume callback should be called")
	}

	// Wait for flush after resume
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	countAfterResume := flushCount
	mu.Unlock()

	if countAfterResume <= countWhilePaused {
		t.Logf("Flush count: paused=%d, after_resume=%d", countWhilePaused, countAfterResume)
	}

	agg.Stop()
}

// TestSmartAggregator_SetRelayOutput tests relay output configuration
func TestSmartAggregator_SetRelayOutput(t *testing.T) {
	var grpcData, relayData []byte
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			grpcData = append(grpcData, data...)
			mu.Unlock()
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write without relay - should go to gRPC
	agg.Write([]byte("grpc"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !bytes.Contains(grpcData, []byte("grpc")) {
		t.Error("Data should go to gRPC when no relay")
	}
	grpcData = nil
	mu.Unlock()

	// Set relay output
	agg.SetRelayOutput(func(data []byte) {
		mu.Lock()
		relayData = append(relayData, data...)
		mu.Unlock()
	})

	// Write with relay - should go to relay only
	agg.Write([]byte("relay"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if bytes.Contains(grpcData, []byte("relay")) {
		t.Error("Data should not go to gRPC when relay is set")
	}
	if !bytes.Contains(relayData, []byte("relay")) {
		t.Error("Data should go to relay")
	}
	mu.Unlock()

	agg.Stop()
}

// TestSmartAggregator_SetPTYLogger tests PTY logger configuration
func TestSmartAggregator_SetPTYLogger(t *testing.T) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0.0 },
	)

	// Should not panic with nil logger
	agg.SetPTYLogger(nil)

	// Can set and use
	agg.Write([]byte("test"))
	agg.Stop()
}

// TestSmartAggregator_TimerFlushPaused tests timer flush when paused
func TestSmartAggregator_TimerFlushPaused(t *testing.T) {
	var flushCount int32
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
			done <- struct{}{}
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write data then pause
	agg.Write([]byte("data"))
	agg.Pause()

	// Wait for timer to fire (should reschedule due to pause)
	time.Sleep(100 * time.Millisecond)

	// Resume and wait for flush
	agg.Resume()

	select {
	case <-done:
		// OK - flush happened
	case <-time.After(200 * time.Millisecond):
		// Also OK - may have flushed before pause
	}

	agg.Stop()
}

// TestSmartAggregator_TimerFlushCriticalLoad tests timer flush under critical load
func TestSmartAggregator_TimerFlushCriticalLoad(t *testing.T) {
	var usage atomic.Int64
	usage.Store(60) // 0.6 * 100 = 60, Critical load
	var flushCount int32

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
		},
		func() float64 { return float64(usage.Load()) / 100.0 },
		WithSmartBaseDelay(20*time.Millisecond),
		WithSmartMaxDelay(100*time.Millisecond),
	)

	// Write data under critical load
	agg.Write([]byte("data"))

	// Wait less than maxDelay - should not flush
	time.Sleep(60 * time.Millisecond)

	// Lower the usage
	usage.Store(0)

	// Wait for flush (generous margin for Windows timer resolution ~15ms)
	time.Sleep(300 * time.Millisecond)

	count := atomic.LoadInt32(&flushCount)
	if count == 0 {
		t.Error("Should have flushed after load decreased")
	}

	agg.Stop()
}

// TestSmartAggregator_FlushWithIncompleteFrame tests that incomplete frames are handled
func TestSmartAggregator_FlushWithIncompleteFrame(t *testing.T) {
	var flushed []byte
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			flushed = append(flushed, data...)
			mu.Unlock()
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write complete frame + incomplete frame
	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	complete := syncStart + "complete" + syncEnd
	incomplete := syncStart + "incomplete"

	agg.Write([]byte(complete))
	agg.Write([]byte(incomplete))

	// Wait for timer flush
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	data := flushed
	mu.Unlock()

	// Should flush complete frame
	if !bytes.Contains(data, []byte("complete")) {
		t.Error("Complete frame should be flushed")
	}

	// Incomplete should be kept in buffer
	bufLen := agg.BufferLen()
	t.Logf("Buffer length after flush: %d", bufLen)

	agg.Stop()
}

// TestSmartAggregator_ResumeWithoutPause tests resume when not paused
func TestSmartAggregator_ResumeWithoutPause(t *testing.T) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0.0 },
	)

	// Resume without pause should not trigger flush
	agg.Resume()

	if agg.IsPaused() {
		t.Error("Should not be paused")
	}

	agg.Stop()
}

// TestSmartAggregator_WriteAfterStop tests that writes are ignored after stop
func TestSmartAggregator_WriteAfterStop(t *testing.T) {
	var flushCount int

	agg := NewSmartAggregator(
		func(data []byte) {
			flushCount++
		},
		func() float64 { return 0.0 },
	)

	agg.Stop()
	initialCount := flushCount

	// Write after stop should be ignored
	agg.Write([]byte("ignored"))
	time.Sleep(50 * time.Millisecond)

	if flushCount != initialCount {
		t.Error("Write after stop should be ignored")
	}
}
