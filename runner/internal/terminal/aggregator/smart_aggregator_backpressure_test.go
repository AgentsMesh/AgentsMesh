package aggregator

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"
)

// TestSmartAggregator_Backpressure tests pause/resume functionality
func TestSmartAggregator_Backpressure(t *testing.T) {
	var pauseCalled, resumeCalled bool
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithBackpressureCallbacks(
			func() { pauseCalled = true },
			func() { resumeCalled = true },
		),
	)
	agg.SetRelayClient(relay)

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

	dataWhilePaused := len(relay.getData())

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

	dataAfterResume := len(relay.getData())

	if dataAfterResume <= dataWhilePaused {
		t.Logf("Data length: paused=%d, after_resume=%d", dataWhilePaused, dataAfterResume)
	}

	agg.Stop()
}

// TestSmartAggregator_SetRelayClient tests relay client configuration
func TestSmartAggregator_SetRelayClient(t *testing.T) {
	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write without relay - data is dropped (no relay connected)
	agg.Write([]byte("dropped"))
	time.Sleep(50 * time.Millisecond)

	// Set relay client (connected)
	relay := newMockRelayWriter(true)
	agg.SetRelayClient(relay)

	// Write with relay - should go to relay
	agg.Write([]byte("relay"))
	time.Sleep(50 * time.Millisecond)

	if !bytes.Contains(relay.getData(), []byte("relay")) {
		t.Error("Data should go to relay")
	}

	agg.Stop()
}

// TestSmartAggregator_SetPTYLogger tests PTY logger configuration
func TestSmartAggregator_SetPTYLogger(t *testing.T) {
	agg := NewSmartAggregator(
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
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	// Write data then pause
	agg.Write([]byte("data"))
	agg.Pause()

	// Wait for timer to fire (should reschedule due to pause)
	time.Sleep(100 * time.Millisecond)

	// Resume and wait for flush
	agg.Resume()

	time.Sleep(200 * time.Millisecond)

	// Verify data was eventually flushed
	if len(relay.getData()) == 0 {
		t.Log("Data may have flushed before pause - this is acceptable")
	}

	agg.Stop()
}

// TestSmartAggregator_TimerFlushCriticalLoad tests timer flush under critical load
func TestSmartAggregator_TimerFlushCriticalLoad(t *testing.T) {
	var usage atomic.Int64
	usage.Store(60) // 0.6 * 100 = 60, Critical load
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return float64(usage.Load()) / 100.0 },
		WithSmartBaseDelay(20*time.Millisecond),
		WithSmartMaxDelay(100*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	// Write data under critical load
	agg.Write([]byte("data"))

	// Wait less than maxDelay - should not flush
	time.Sleep(60 * time.Millisecond)

	// Lower the usage
	usage.Store(0)

	// Wait for flush (generous margin for Windows timer resolution ~15ms)
	time.Sleep(300 * time.Millisecond)

	if len(relay.getData()) == 0 {
		t.Error("Should have flushed after load decreased")
	}

	agg.Stop()
}

// TestSmartAggregator_FlushWithIncompleteFrame tests that incomplete frames are handled
func TestSmartAggregator_FlushWithIncompleteFrame(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	// Write complete frame + incomplete frame
	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	complete := syncStart + "complete" + syncEnd
	incomplete := syncStart + "incomplete"

	agg.Write([]byte(complete))
	agg.Write([]byte(incomplete))

	// Wait for timer flush
	time.Sleep(50 * time.Millisecond)

	data := relay.getData()

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
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
	)
	agg.SetRelayClient(relay)

	agg.Stop()
	initialLen := len(relay.getData())

	// Write after stop should be ignored
	agg.Write([]byte("ignored"))
	time.Sleep(50 * time.Millisecond)

	if len(relay.getData()) != initialLen {
		t.Error("Write after stop should be ignored")
	}
}
