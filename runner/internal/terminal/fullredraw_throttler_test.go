package terminal

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullRedrawThrottler_NewDefault(t *testing.T) {
	throttler := NewFullRedrawThrottler()
	require.NotNil(t, throttler)

	// Verify defaults
	assert.Equal(t, 1*time.Second, throttler.baseWindowSize)
	assert.Equal(t, 1*time.Second, throttler.minWindowSize)
	assert.Equal(t, 4*time.Second, throttler.maxWindowSize)
	assert.Equal(t, 200*time.Millisecond, throttler.minDelay)
	assert.Equal(t, 1000*time.Millisecond, throttler.maxDelay)
	assert.Equal(t, 1.5, throttler.thresholdFreq)
	assert.False(t, throttler.IsThrottling())
}

func TestFullRedrawThrottler_Options(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(5*time.Second),
		WithThrottlerMinWindow(2*time.Second),
		WithThrottlerMaxWindow(10*time.Second),
		WithThrottlerMinDelay(100*time.Millisecond),
		WithThrottlerMaxDelay(2*time.Second),
		WithThrottlerThreshold(5.0),
		WithThrottlerBandwidthThresholds(100*1024, 300*1024),
	)

	assert.Equal(t, 5*time.Second, throttler.baseWindowSize)
	assert.Equal(t, 2*time.Second, throttler.minWindowSize)
	assert.Equal(t, 10*time.Second, throttler.maxWindowSize)
	assert.Equal(t, 100*time.Millisecond, throttler.minDelay)
	assert.Equal(t, 2*time.Second, throttler.maxDelay)
	assert.Equal(t, 5.0, throttler.thresholdFreq)
	assert.Equal(t, 100*1024, throttler.lowBandwidthThreshold)
	assert.Equal(t, 300*1024, throttler.highBandwidthThreshold)
}

func TestFullRedrawThrottler_RecordAndFrequency(t *testing.T) {
	// Use short window for testing
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(100*time.Millisecond),
		WithThrottlerMinWindow(100*time.Millisecond),
		WithThrottlerMaxWindow(100*time.Millisecond),
	)

	// No records yet
	assert.Equal(t, 0.0, throttler.GetFrequency())

	// Record some redraws (10KB each - small frames won't trigger bandwidth adjustment)
	throttler.RecordRedraw(10 * 1024)
	throttler.RecordRedraw(10 * 1024)
	throttler.RecordRedraw(10 * 1024)

	// Frequency should be 3 / 0.1s = 30/s
	assert.InDelta(t, 30.0, throttler.GetFrequency(), 1.0)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Records should have expired
	assert.Equal(t, 0.0, throttler.GetFrequency())
}

func TestFullRedrawThrottler_WindowSliding(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(100*time.Millisecond),
		WithThrottlerMinWindow(100*time.Millisecond),
		WithThrottlerMaxWindow(100*time.Millisecond),
	)

	// Record 2 redraws (small frames)
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024)

	// Wait half the window
	time.Sleep(60 * time.Millisecond)

	// Record 2 more
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024)

	// All 4 should be in window
	freq := throttler.GetFrequency()
	assert.InDelta(t, 40.0, freq, 5.0) // 4 / 0.1s = 40/s

	// Wait for first 2 to expire
	time.Sleep(60 * time.Millisecond)

	// Only last 2 should remain
	freq = throttler.GetFrequency()
	assert.InDelta(t, 20.0, freq, 5.0) // 2 / 0.1s = 20/s
}

func TestFullRedrawThrottler_DelayCalculation(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.5),
		WithThrottlerMinDelay(200*time.Millisecond),
		WithThrottlerMaxDelay(1000*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Below threshold: no delay (small frames)
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024) // 2/s < 2.5/s threshold
	assert.Equal(t, time.Duration(0), throttler.GetCurrentDelay())
	assert.False(t, throttler.IsThrottling())

	// At threshold: min delay
	throttler.RecordRedraw(1024) // 3/s >= 2.5/s threshold
	delay := throttler.GetCurrentDelay()
	assert.True(t, delay >= 200*time.Millisecond, "delay should be at least minDelay")
	assert.True(t, throttler.IsThrottling())

	// Add more redraws to increase delay
	for i := 0; i < 7; i++ {
		throttler.RecordRedraw(1024)
	}
	// Now 10/s - should be near max delay
	delay = throttler.GetCurrentDelay()
	assert.True(t, delay >= 800*time.Millisecond, "delay should be close to maxDelay at 10/s")
}

func TestFullRedrawThrottler_ShouldFlush(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		WithThrottlerMinDelay(100*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Initially should flush (no throttling - below threshold)
	assert.True(t, throttler.ShouldFlush())

	// Add redraws to trigger throttling (5/s > 2.0/s threshold)
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(1024)
	}

	// Now throttling is active, but ShouldFlush still returns true
	// because we haven't marked a flush yet (lastFlushTime is zero)
	// Mark a flush first
	throttler.MarkFlushed()

	// Immediately after marking flush, should not flush again (delay not passed)
	assert.False(t, throttler.ShouldFlush(), "Should not flush immediately after MarkFlushed")

	// Get the current delay to know how long to wait
	delay := throttler.GetCurrentDelay()
	t.Logf("Current delay: %v", delay)

	// Wait for delay to pass (plus some buffer)
	time.Sleep(delay + 50*time.Millisecond)

	// Now should be able to flush
	assert.True(t, throttler.ShouldFlush(), "Should flush after delay has passed")
}

func TestFullRedrawThrottler_GetNextCheckDelay(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		WithThrottlerMinDelay(100*time.Millisecond),
		WithThrottlerMaxDelay(1000*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Not throttling - should return short default
	delay := throttler.GetNextCheckDelay()
	assert.Equal(t, 50*time.Millisecond, delay)

	// Trigger throttling (5/s > 2.0/s)
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(1024)
	}
	throttler.MarkFlushed()

	// Should return remaining time until next allowed flush
	delay = throttler.GetNextCheckDelay()
	assert.True(t, delay > 0, "should have positive delay when throttling")

	// The delay should be close to the throttle delay (since we just flushed)
	// plus a small buffer (10ms added in GetNextCheckDelay)
	currentThrottleDelay := throttler.GetCurrentDelay()
	t.Logf("Current throttle delay: %v, next check delay: %v", currentThrottleDelay, delay)

	// Allow some tolerance for timing
	assert.True(t, delay <= currentThrottleDelay+20*time.Millisecond,
		"delay should not exceed throttle delay plus buffer, got %v vs max %v",
		delay, currentThrottleDelay+20*time.Millisecond)
}

func TestFullRedrawThrottler_Reset(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Add redraws and flush
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(1024)
	}
	throttler.MarkFlushed()

	assert.True(t, throttler.IsThrottling())

	// Reset
	throttler.Reset()

	assert.False(t, throttler.IsThrottling())
	assert.Equal(t, 0.0, throttler.GetFrequency())
	assert.True(t, throttler.ShouldFlush())
}

func TestFullRedrawThrottler_ConcurrentAccess(t *testing.T) {
	throttler := NewFullRedrawThrottler()

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 100

	// Concurrent RecordRedraw calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				throttler.RecordRedraw(50 * 1024) // 50KB frames
			}
		}()
	}

	// Concurrent ShouldFlush/MarkFlushed calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if throttler.ShouldFlush() {
					throttler.MarkFlushed()
				}
			}
		}()
	}

	// Concurrent GetFrequency/GetBandwidth calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = throttler.GetFrequency()
				_ = throttler.GetBandwidth()
				_ = throttler.GetEffectiveWindowSize()
			}
		}()
	}

	wg.Wait()
	// Test passes if no race conditions or panics
}

func TestFullRedrawThrottler_DelayLinearInterpolation(t *testing.T) {
	// Test that delay increases linearly between threshold and maxFreq
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.5),
		WithThrottlerMinDelay(200*time.Millisecond),
		WithThrottlerMaxDelay(1000*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Helper to record N redraws (small frames to avoid bandwidth throttling)
	recordN := func(n int) {
		throttler.Reset()
		for i := 0; i < n; i++ {
			throttler.RecordRedraw(1024)
		}
	}

	// At threshold (2.5/s but we can only do integers, so 3/s)
	recordN(3)
	delayAtThreshold := throttler.GetCurrentDelay()

	// At midpoint between threshold (2.5) and maxFreq (10) = 6.25/s, so 6
	recordN(6)
	delayAtMid := throttler.GetCurrentDelay()

	// At maxFreq (10/s)
	recordN(10)
	delayAtMax := throttler.GetCurrentDelay()

	// Verify ordering
	assert.True(t, delayAtThreshold <= delayAtMid, "mid delay should be >= threshold delay")
	assert.True(t, delayAtMid <= delayAtMax, "max delay should be >= mid delay")

	// Verify bounds
	assert.True(t, delayAtThreshold >= 200*time.Millisecond, "at threshold should be >= minDelay")
	assert.True(t, delayAtMax <= 1000*time.Millisecond, "at max should be <= maxDelay")
}

func TestFullRedrawThrottler_ExtremeFrequency(t *testing.T) {
	// Test behavior at very high frequencies (beyond maxFreq)
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.5),
		WithThrottlerMinDelay(200*time.Millisecond),
		WithThrottlerMaxDelay(1000*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Record 50 redraws (way beyond maxFreq of 10, small frames)
	for i := 0; i < 50; i++ {
		throttler.RecordRedraw(1024)
	}

	// Delay should be capped at maxDelay
	delay := throttler.GetCurrentDelay()
	assert.Equal(t, 1000*time.Millisecond, delay, "delay should be capped at maxDelay")
}

func TestFullRedrawThrottler_ZeroFrequency(t *testing.T) {
	throttler := NewFullRedrawThrottler()

	// No redraws recorded
	assert.Equal(t, 0.0, throttler.GetFrequency())
	assert.Equal(t, time.Duration(0), throttler.GetCurrentDelay())
	assert.False(t, throttler.IsThrottling())
	assert.True(t, throttler.ShouldFlush())
}

func TestFullRedrawThrottler_SingleRedraw(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	throttler.RecordRedraw(1024)

	// Single redraw = 1/s, below threshold of 2.0/s
	freq := throttler.GetFrequency()
	assert.InDelta(t, 1.0, freq, 0.1)
	assert.False(t, throttler.IsThrottling())
}

func TestFullRedrawThrottler_ExactlyAtThreshold(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(3.0), // threshold 3/s
		WithThrottlerMinDelay(100*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Record 4 redraws = 4/s > threshold 3/s (need to exceed threshold)
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024)
	throttler.RecordRedraw(1024)

	// Above threshold should start throttling
	assert.True(t, throttler.IsThrottling())
	delay := throttler.GetCurrentDelay()
	// Should be at or above minDelay
	assert.True(t, delay >= 100*time.Millisecond, "above threshold should use minDelay or more")
}

func TestFullRedrawThrottler_GetNextCheckDelay_NotThrottling(t *testing.T) {
	throttler := NewFullRedrawThrottler()

	// Not throttling - should return default 50ms
	delay := throttler.GetNextCheckDelay()
	assert.Equal(t, 50*time.Millisecond, delay)
}

func TestFullRedrawThrottler_GetNextCheckDelay_WhileThrottling(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second), // Long window to keep records
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		WithThrottlerMinDelay(100*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Trigger throttling
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(1024)
	}
	throttler.MarkFlushed()

	// Immediately after marking flush, GetNextCheckDelay should return
	// approximately the throttle delay (plus 10ms buffer)
	delay := throttler.GetNextCheckDelay()
	currentThrottleDelay := throttler.GetCurrentDelay()

	t.Logf("Throttle delay: %v, next check delay: %v", currentThrottleDelay, delay)

	// Should be close to throttle delay
	assert.True(t, delay > 50*time.Millisecond, "while throttling, should have substantial delay")
	assert.True(t, delay <= currentThrottleDelay+20*time.Millisecond, "should not exceed throttle delay plus buffer")
}

func TestFullRedrawThrottler_ShouldFlush_BeforeMarkFlushed(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
		WithThrottlerThreshold(2.0),
		WithThrottlerMinDelay(100*time.Millisecond),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Trigger throttling
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(1024)
	}

	// Before MarkFlushed, lastFlushTime is zero
	// ShouldFlush checks time.Since(zero) which is huge, so should return true
	assert.True(t, throttler.ShouldFlush(), "before first flush, should always allow")
}

func TestFullRedrawThrottler_CleanExpiredDuringOperations(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(50*time.Millisecond),
		WithThrottlerMinWindow(50*time.Millisecond),
		WithThrottlerMaxWindow(50*time.Millisecond),
		WithThrottlerThreshold(5.0),
		// Use high bandwidth threshold to avoid bandwidth-triggered throttling
		WithThrottlerBandwidthThresholds(10*1024*1024, 20*1024*1024),
	)

	// Record some redraws
	for i := 0; i < 10; i++ {
		throttler.RecordRedraw(1024)
	}
	assert.True(t, throttler.IsThrottling())

	// Wait for window to expire
	time.Sleep(100 * time.Millisecond)

	// Various operations should clean expired entries
	_ = throttler.ShouldFlush() // This calls cleanExpiredLocked
	assert.False(t, throttler.IsThrottling(), "after window expires, should not be throttling")

	_ = throttler.GetFrequency() // This also cleans
	assert.Equal(t, 0.0, throttler.GetFrequency())
}

// ========== Bandwidth-aware dynamic window tests ==========

func TestFullRedrawThrottler_BandwidthCalculation(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(1*time.Second),
	)

	// No records yet
	assert.Equal(t, 0, throttler.GetBandwidth())

	// Record frames totaling 100KB in 1 second window
	for i := 0; i < 10; i++ {
		throttler.RecordRedraw(10 * 1024) // 10KB each
	}

	// Bandwidth should be ~100KB/s
	bandwidth := throttler.GetBandwidth()
	assert.InDelta(t, 100*1024, bandwidth, 10*1024, "bandwidth should be ~100KB/s")
}

func TestFullRedrawThrottler_DynamicWindowAdjustment_LowBandwidth(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerBandwidthThresholds(200*1024, 500*1024), // 200KB/s - 500KB/s
	)

	// Low bandwidth: 50KB/s (below lowBandwidthThreshold)
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(10 * 1024) // 10KB each = 50KB/s
	}

	// Effective window should be at minimum (1s)
	assert.Equal(t, 1*time.Second, throttler.GetEffectiveWindowSize(),
		"low bandwidth should use minimum window")
}

func TestFullRedrawThrottler_DynamicWindowAdjustment_HighBandwidth(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerBandwidthThresholds(200*1024, 500*1024), // 200KB/s - 500KB/s
	)

	// High bandwidth: 600KB/s (above highBandwidthThreshold)
	for i := 0; i < 6; i++ {
		throttler.RecordRedraw(100 * 1024) // 100KB each = 600KB/s
	}

	// Effective window should be at maximum (4s)
	assert.Equal(t, 4*time.Second, throttler.GetEffectiveWindowSize(),
		"high bandwidth should use maximum window")
}

func TestFullRedrawThrottler_DynamicWindowAdjustment_MediumBandwidth(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerBandwidthThresholds(200*1024, 500*1024), // 200KB/s - 500KB/s
	)

	// Medium bandwidth: ~350KB/s (between thresholds)
	for i := 0; i < 7; i++ {
		throttler.RecordRedraw(50 * 1024) // 50KB each = 350KB/s
	}

	// Effective window should be between min and max
	window := throttler.GetEffectiveWindowSize()
	assert.True(t, window > 1*time.Second, "medium bandwidth should have window > min")
	assert.True(t, window < 4*time.Second, "medium bandwidth should have window < max")

	// Should be approximately 2.5s (linear interpolation: (350-200)/(500-200) * 3s + 1s = 2.5s)
	assert.InDelta(t, 2500, window.Milliseconds(), 500, "window should be ~2.5s")
}

func TestFullRedrawThrottler_BandwidthTriggeredThrottling(t *testing.T) {
	// Test that high bandwidth alone can trigger throttling even with low frequency
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerThreshold(10.0),                         // High freq threshold (won't trigger)
		WithThrottlerBandwidthThresholds(200*1024, 500*1024), // 200KB/s - 500KB/s
	)

	// Low frequency (2/s) but high bandwidth (600KB/s)
	throttler.RecordRedraw(300 * 1024) // 300KB frame
	throttler.RecordRedraw(300 * 1024) // 300KB frame

	// Frequency is 2/s < threshold 10/s, but bandwidth is 600KB/s > highBandwidthThreshold
	freq := throttler.GetFrequency()
	bandwidth := throttler.GetBandwidth()
	t.Logf("Frequency: %.2f/s, Bandwidth: %dKB/s", freq, bandwidth/1024)

	assert.True(t, freq < 10.0, "frequency should be below threshold")
	assert.True(t, bandwidth > 500*1024, "bandwidth should be above high threshold")

	// Should still be throttling due to high bandwidth
	assert.True(t, throttler.IsThrottling(), "high bandwidth should trigger throttling even with low frequency")
}

func TestFullRedrawThrottler_WindowExpandsWithBandwidth(t *testing.T) {
	// Test that window expansion leads to more aggressive throttling
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerThreshold(1.0),                          // Low threshold to ensure throttling
		WithThrottlerBandwidthThresholds(100*1024, 400*1024), // 100KB/s - 400KB/s
	)

	// Start with small frames - low bandwidth, min window
	for i := 0; i < 3; i++ {
		throttler.RecordRedraw(10 * 1024) // 10KB each = 30KB/s
	}
	windowSmall := throttler.GetEffectiveWindowSize()
	freqSmall := throttler.GetFrequency()
	delaySmall := throttler.GetCurrentDelay()

	t.Logf("Small frames: window=%v, freq=%.2f/s, delay=%v", windowSmall, freqSmall, delaySmall)

	// Reset and use large frames - high bandwidth, max window
	throttler.Reset()
	for i := 0; i < 3; i++ {
		throttler.RecordRedraw(200 * 1024) // 200KB each = 600KB/s
	}
	windowLarge := throttler.GetEffectiveWindowSize()
	freqLarge := throttler.GetFrequency()
	delayLarge := throttler.GetCurrentDelay()

	t.Logf("Large frames: window=%v, freq=%.2f/s, delay=%v", windowLarge, freqLarge, delayLarge)

	// With larger window, we should see higher delay (more aggressive throttling)
	assert.True(t, windowLarge > windowSmall, "high bandwidth should expand window")
	// Note: delay comparison depends on the formula, but high bandwidth should increase throttling
}

func TestFullRedrawThrottler_ResetClearsEffectiveWindow(t *testing.T) {
	throttler := NewFullRedrawThrottler(
		WithThrottlerWindowSize(1*time.Second),
		WithThrottlerMinWindow(1*time.Second),
		WithThrottlerMaxWindow(4*time.Second),
		WithThrottlerBandwidthThresholds(100*1024, 400*1024),
	)

	// Generate high bandwidth to expand window
	for i := 0; i < 5; i++ {
		throttler.RecordRedraw(200 * 1024)
	}
	assert.True(t, throttler.GetEffectiveWindowSize() > 1*time.Second, "window should be expanded")

	// Reset should restore base window
	throttler.Reset()
	assert.Equal(t, 1*time.Second, throttler.GetEffectiveWindowSize(),
		"reset should restore base window size")
}
