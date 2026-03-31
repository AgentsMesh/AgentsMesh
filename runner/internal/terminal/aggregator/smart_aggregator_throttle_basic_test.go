package aggregator

import (
	"bytes"
	"testing"
	"time"
)

// TestSmartAggregator_FullRedrawThrottling_Enabled tests that throttling reduces flush rate
func TestSmartAggregator_FullRedrawThrottling_Enabled(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithFullRedrawThrottling(
			WithThrottlerWindowSize(500*time.Millisecond),
			WithThrottlerMinWindow(500*time.Millisecond),
			WithThrottlerMaxWindow(500*time.Millisecond),
			WithThrottlerThreshold(2.0), // 2/s = 1 in 500ms window
			WithThrottlerMinDelay(100*time.Millisecond),
		),
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	// Send multiple full redraw frames rapidly
	for i := 0; i < 10; i++ {
		frame := buildFullRedrawFrame("frame content " + string(rune('0'+i)))
		agg.Write(frame)
		time.Sleep(20 * time.Millisecond) // 50 fps - should trigger throttling
	}

	// Wait for any pending flushes
	time.Sleep(200 * time.Millisecond)

	count := int32(relay.sendCount())

	// With throttling, we should have significantly fewer flushes than frames
	// Without throttling, we'd have ~10 flushes (one per frame)
	// With throttling (100ms min delay), we should have ~2-5 flushes in 200ms total
	t.Logf("Full redraw frames written: 10, actual flushes: %d", count)

	if count >= 10 {
		t.Errorf("Throttling did not reduce flush rate: %d flushes for 10 frames", count)
	}

	// Verify we still get the latest data
	data := relay.getData()
	if !bytes.Contains(data, []byte("frame content")) {
		t.Error("Latest frame data should be preserved")
	}
}

// TestSmartAggregator_FullRedrawThrottling_IncrementalNotThrottled tests that
// incremental frames are not throttled
func TestSmartAggregator_FullRedrawThrottling_IncrementalNotThrottled(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
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
	agg.SetRelayClient(relay)
	defer agg.Stop()

	// Send incremental frames (small, no clear screen)
	for i := 0; i < 5; i++ {
		frame := buildSyncFrame("small") // Small incremental frame
		agg.Write(frame)
		time.Sleep(30 * time.Millisecond)
	}

	// Wait for flushes
	time.Sleep(100 * time.Millisecond)

	count := relay.sendCount()

	// Incremental frames should not be throttled (or minimally affected)
	// With 30ms between writes and 10ms base delay, we should get ~4-5 flushes
	t.Logf("Incremental frames written: 5, actual flushes: %d", count)

	if count < 3 {
		t.Errorf("Incremental frames should not be heavily throttled: only %d flushes for 5 frames", count)
	}
}

// TestSmartAggregator_FullRedrawThrottling_Disabled tests behavior without throttling
func TestSmartAggregator_FullRedrawThrottling_Disabled(t *testing.T) {
	relay := newMockRelayWriter(true)

	// Create aggregator WITHOUT throttling
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(10*time.Millisecond),
		// No WithFullRedrawThrottling
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	// Send full redraw frames rapidly
	for i := 0; i < 5; i++ {
		frame := buildFullRedrawFrame("frame content")
		agg.Write(frame)
		time.Sleep(30 * time.Millisecond)
	}

	// Wait for flushes
	time.Sleep(100 * time.Millisecond)

	count := relay.sendCount()

	// Without throttling, we should get ~4-5 flushes (one per frame, aggregated by delay)
	t.Logf("Frames written: 5, actual flushes without throttling: %d", count)

	// Should have at least 3 flushes (no throttling effect)
	if count < 3 {
		t.Errorf("Without throttling, should have more flushes: only %d for 5 frames", count)
	}
}
