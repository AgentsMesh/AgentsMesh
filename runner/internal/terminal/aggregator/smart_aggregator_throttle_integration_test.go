//go:build integration

package aggregator

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestSmartAggregator_FullRedrawThrottling_MixedFrames tests behavior with
// alternating full redraw and incremental frames
func TestSmartAggregator_FullRedrawThrottling_MixedFrames(t *testing.T) {
	var mu sync.Mutex
	var flushedData [][]byte

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			flushedData = append(flushedData, dataCopy)
			mu.Unlock()
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(500*time.Millisecond),
			WithThrottlerMinWindow(500*time.Millisecond),
			WithThrottlerMaxWindow(500*time.Millisecond),
			WithThrottlerThreshold(3.0),
			WithThrottlerMinDelay(100*time.Millisecond),
		),
	)
	defer agg.Stop()

	// Alternate between full redraw and incremental frames
	for i := 0; i < 6; i++ {
		if i%2 == 0 {
			// Full redraw frame
			frame := buildFullRedrawFrame("full_" + string(rune('0'+i)))
			agg.Write(frame)
		} else {
			// Incremental frame (small)
			frame := buildSyncFrame("inc_" + string(rune('0'+i)))
			agg.Write(frame)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Wait and force flush
	time.Sleep(150 * time.Millisecond)
	agg.Flush()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Verify we got both types of content
	hasFullRedraw := false
	hasIncremental := false
	for _, data := range flushedData {
		if bytes.Contains(data, []byte("full_")) {
			hasFullRedraw = true
		}
		if bytes.Contains(data, []byte("inc_")) {
			hasIncremental = true
		}
	}

	t.Logf("Mixed frames: %d flushes, hasFullRedraw=%v, hasIncremental=%v",
		len(flushedData), hasFullRedraw, hasIncremental)

	// Both types should be delivered eventually
	if !hasFullRedraw {
		t.Error("Full redraw frames should be delivered")
	}
	if !hasIncremental {
		t.Error("Incremental frames should be delivered")
	}
}

// TestSmartAggregator_FullRedrawThrottling_StopForceFlush tests that Stop()
// flushes all pending data regardless of throttling
func TestSmartAggregator_FullRedrawThrottling_StopForceFlush(t *testing.T) {
	var mu sync.Mutex
	var flushedData [][]byte

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			flushedData = append(flushedData, dataCopy)
			mu.Unlock()
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(1*time.Second),
			WithThrottlerMinWindow(1*time.Second),
			WithThrottlerMaxWindow(1*time.Second),
			WithThrottlerThreshold(1.0),
			WithThrottlerMinDelay(500*time.Millisecond), // Long delay
		),
	)

	// Write frames rapidly to trigger throttling
	for i := 0; i < 5; i++ {
		frame := buildFullRedrawFrame("stop_test_" + string(rune('0'+i)))
		agg.Write(frame)
		time.Sleep(5 * time.Millisecond)
	}

	// Stop immediately - should force flush all pending data
	agg.Stop()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have flushed something
	totalBytes := 0
	for _, data := range flushedData {
		totalBytes += len(data)
	}

	t.Logf("Stop force flush: %d flushes, %d total bytes", len(flushedData), totalBytes)

	if len(flushedData) == 0 {
		t.Error("Stop() should force flush pending data")
	}
}

// TestSmartAggregator_FullRedrawThrottling_WithBackpressure tests interaction
// between throttling and backpressure
func TestSmartAggregator_FullRedrawThrottling_WithBackpressure(t *testing.T) {
	var flushCount int32

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(500*time.Millisecond),
			WithThrottlerMinWindow(500*time.Millisecond),
			WithThrottlerMaxWindow(500*time.Millisecond),
			WithThrottlerThreshold(2.0),
			WithThrottlerMinDelay(50*time.Millisecond),
		),
	)
	defer agg.Stop()

	// Pause the aggregator (backpressure)
	agg.Pause()

	// Write frames while paused
	for i := 0; i < 5; i++ {
		frame := buildFullRedrawFrame("backpressure_" + string(rune('0'+i)))
		agg.Write(frame)
		time.Sleep(10 * time.Millisecond)
	}

	// Count before resume
	countBeforeResume := atomic.LoadInt32(&flushCount)
	t.Logf("Before resume: %d flushes", countBeforeResume)

	// Resume
	agg.Resume()

	// Wait for pending data to flush
	time.Sleep(200 * time.Millisecond)

	countAfterResume := atomic.LoadInt32(&flushCount)
	t.Logf("After resume: %d flushes", countAfterResume)

	// Should have flushed after resume
	if countAfterResume <= countBeforeResume {
		t.Error("Should have flushed after resume")
	}
}

// TestSmartAggregator_FullRedrawThrottling_WithQueuePressure tests throttling
// behavior when queue has pressure
func TestSmartAggregator_FullRedrawThrottling_WithQueuePressure(t *testing.T) {
	var flushCount int32
	queueUsage := float64(0)
	var queueMu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
		},
		func() float64 {
			queueMu.Lock()
			defer queueMu.Unlock()
			return queueUsage
		},
		WithSmartBaseDelay(10*time.Millisecond),
		WithSmartMaxDelay(100*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(500*time.Millisecond),
			WithThrottlerMinWindow(500*time.Millisecond),
			WithThrottlerMaxWindow(500*time.Millisecond),
			WithThrottlerThreshold(3.0),
			WithThrottlerMinDelay(50*time.Millisecond),
		),
	)
	defer agg.Stop()

	// Start with low queue pressure
	queueMu.Lock()
	queueUsage = 0.1
	queueMu.Unlock()

	// Write some frames
	for i := 0; i < 3; i++ {
		frame := buildFullRedrawFrame("low_pressure_" + string(rune('0'+i)))
		agg.Write(frame)
		time.Sleep(30 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	countLowPressure := atomic.LoadInt32(&flushCount)

	// Increase queue pressure
	queueMu.Lock()
	queueUsage = 0.8 // High pressure
	queueMu.Unlock()

	// Write more frames under high pressure
	for i := 0; i < 3; i++ {
		frame := buildFullRedrawFrame("high_pressure_" + string(rune('0'+i)))
		agg.Write(frame)
		time.Sleep(30 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)
	countHighPressure := atomic.LoadInt32(&flushCount)

	t.Logf("Low pressure flushes: %d, High pressure total: %d",
		countLowPressure, countHighPressure)

	// Under high pressure, aggregator uses longer delays, so fewer flushes expected
	// This test just verifies that both mechanisms work together without crashing
	if countHighPressure < countLowPressure {
		t.Error("Should have at least as many flushes after more writes")
	}
}

// TestSmartAggregator_FullRedrawThrottling_PlainText tests that plain text
// (without sync frames) is not affected by throttling
func TestSmartAggregator_FullRedrawThrottling_PlainText(t *testing.T) {
	var flushCount int32
	var mu sync.Mutex
	var totalBytes int

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
			mu.Lock()
			totalBytes += len(data)
			mu.Unlock()
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(500*time.Millisecond),
			WithThrottlerMinWindow(500*time.Millisecond),
			WithThrottlerMaxWindow(500*time.Millisecond),
			WithThrottlerThreshold(2.0),
			WithThrottlerMinDelay(200*time.Millisecond),
		),
	)
	defer agg.Stop()

	// Write plain text (no sync frames)
	for i := 0; i < 5; i++ {
		agg.Write([]byte("plain text line " + string(rune('0'+i)) + "\n"))
		time.Sleep(30 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	count := atomic.LoadInt32(&flushCount)
	mu.Lock()
	bytes := totalBytes
	mu.Unlock()

	t.Logf("Plain text: %d flushes, %d bytes", count, bytes)

	// Plain text has no sync frames, so IsLastFrameFullRedraw returns false
	// Therefore no throttling should apply
	if count < 3 {
		t.Errorf("Plain text should not be throttled: only %d flushes", count)
	}
}
