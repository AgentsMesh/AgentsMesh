// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// redrawRecord stores a redraw event with its size for bandwidth tracking.
type redrawRecord struct {
	time  time.Time
	bytes int
}

// FullRedrawThrottler provides adaptive throttling for high-frequency full-screen redraws.
//
// Key features:
// 1. Frequency-based throttling: reduces transmission when redraw rate exceeds threshold
// 2. Bandwidth-aware dynamic window: adjusts window size based on traffic volume
//
// When bandwidth is high (e.g., 500KB/s of full redraws), the window expands,
// accumulating more records, which increases the calculated frequency and triggers
// more aggressive throttling. This causes intermediate frames to be skipped,
// sending only the latest state.
//
// Example behavior with bandwidth awareness:
//   - Low traffic (<200KB/s): window=1s, normal responsiveness
//   - Medium traffic (200-500KB/s): window=2s, moderate throttling
//   - High traffic (>500KB/s): window=4s, aggressive throttling, only latest frames sent
type FullRedrawThrottler struct {
	mu sync.Mutex

	// Sliding window: redraw records with timestamps and sizes
	redrawRecords []redrawRecord

	// Base window parameters (adjusted dynamically based on bandwidth)
	baseWindowSize time.Duration // Base window size (default 1s)
	minWindowSize  time.Duration // Minimum window size (default 1s)
	maxWindowSize  time.Duration // Maximum window size (default 4s)

	// Current effective window size (dynamically adjusted)
	effectiveWindowSize time.Duration

	// Bandwidth thresholds for window adjustment
	lowBandwidthThreshold  int // Below this (bytes/s), use min window (default 200KB/s)
	highBandwidthThreshold int // Above this (bytes/s), use max window (default 500KB/s)

	// Throttling parameters
	minDelay      time.Duration // Minimum throttle delay (default 200ms)
	maxDelay      time.Duration // Maximum throttle delay (default 1000ms)
	thresholdFreq float64       // Frequency threshold to start throttling (default 1.5/s)

	// State
	lastFlushTime time.Time // Last actual transmission time
}

// FullRedrawThrottlerOption is a functional option for FullRedrawThrottler.
type FullRedrawThrottlerOption func(*FullRedrawThrottler)

// WithThrottlerWindowSize sets the base sliding window size.
// The effective window will be adjusted between minWindow and maxWindow based on bandwidth.
func WithThrottlerWindowSize(d time.Duration) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.baseWindowSize = d
	}
}

// WithThrottlerMinWindow sets the minimum window size (used when bandwidth is low).
func WithThrottlerMinWindow(d time.Duration) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.minWindowSize = d
	}
}

// WithThrottlerMaxWindow sets the maximum window size (used when bandwidth is high).
func WithThrottlerMaxWindow(d time.Duration) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.maxWindowSize = d
	}
}

// WithThrottlerMinDelay sets the minimum throttle delay.
func WithThrottlerMinDelay(d time.Duration) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.minDelay = d
	}
}

// WithThrottlerMaxDelay sets the maximum throttle delay.
func WithThrottlerMaxDelay(d time.Duration) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.maxDelay = d
	}
}

// WithThrottlerThreshold sets the frequency threshold (redraws/second) to start throttling.
func WithThrottlerThreshold(freq float64) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.thresholdFreq = freq
	}
}

// WithThrottlerBandwidthThresholds sets the bandwidth thresholds for window adjustment.
// lowThreshold: below this (bytes/s), use minimum window
// highThreshold: above this (bytes/s), use maximum window
func WithThrottlerBandwidthThresholds(lowThreshold, highThreshold int) FullRedrawThrottlerOption {
	return func(t *FullRedrawThrottler) {
		t.lowBandwidthThreshold = lowThreshold
		t.highBandwidthThreshold = highThreshold
	}
}

// NewFullRedrawThrottler creates a new FullRedrawThrottler with default settings.
//
// Default parameters:
//   - Base window: 1 second
//   - Min window: 1 second (low bandwidth)
//   - Max window: 4 seconds (high bandwidth)
//   - Threshold: 1.5 redraws/second
//   - Min delay: 200ms
//   - Max delay: 1000ms
//   - Low bandwidth threshold: 200KB/s
//   - High bandwidth threshold: 500KB/s
func NewFullRedrawThrottler(opts ...FullRedrawThrottlerOption) *FullRedrawThrottler {
	t := &FullRedrawThrottler{
		baseWindowSize:         1 * time.Second,
		minWindowSize:          1 * time.Second,
		maxWindowSize:          4 * time.Second,
		effectiveWindowSize:    1 * time.Second,
		lowBandwidthThreshold:  200 * 1024,  // 200KB/s
		highBandwidthThreshold: 500 * 1024,  // 500KB/s
		minDelay:               200 * time.Millisecond,
		maxDelay:               1000 * time.Millisecond,
		thresholdFreq:          1.5, // 1.5/s - lower threshold since we have bandwidth awareness
		redrawRecords:          make([]redrawRecord, 0, 64),
	}

	for _, opt := range opts {
		opt(t)
	}

	// Initialize effective window to base
	t.effectiveWindowSize = t.baseWindowSize

	return t
}

// RecordRedraw records a full-screen redraw event with its size.
// Call this when a full redraw frame is detected.
// The frameBytes parameter is used for bandwidth calculation and window adjustment.
func (t *FullRedrawThrottler) RecordRedraw(frameBytes int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.redrawRecords = append(t.redrawRecords, redrawRecord{
		time:  now,
		bytes: frameBytes,
	})

	// Clean expired entries and adjust window based on bandwidth
	t.cleanExpiredLocked(now)
	t.adjustWindowLocked()

	bandwidth := t.getBandwidthLocked()
	logger.Terminal().Debug("FullRedrawThrottler: recorded redraw",
		"frame_bytes", frameBytes,
		"count_in_window", len(t.redrawRecords),
		"frequency", t.getFrequencyLocked(),
		"bandwidth_kbps", bandwidth/1024,
		"effective_window", t.effectiveWindowSize,
		"threshold", t.thresholdFreq)
}

// ShouldFlush determines whether data should be flushed now or throttled.
// Returns true if enough time has passed since last flush (based on adaptive delay).
// Returns false if we're in throttling mode and should skip this flush.
//
// This method does NOT modify state - it only reads. Call MarkFlushed after successful flush.
func (t *FullRedrawThrottler) ShouldFlush() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clean expired entries
	t.cleanExpiredLocked(time.Now())

	delay := t.calculateDelayLocked()
	if delay == 0 {
		// Not throttling
		return true
	}

	// Check if enough time has passed since last flush
	elapsed := time.Since(t.lastFlushTime)
	shouldFlush := elapsed >= delay

	logger.Terminal().Debug("FullRedrawThrottler: shouldFlush check",
		"delay", delay,
		"elapsed", elapsed,
		"should_flush", shouldFlush,
		"bandwidth_kbps", t.getBandwidthLocked()/1024)

	return shouldFlush
}

// MarkFlushed records that a flush was performed.
// Call this after successfully sending data.
func (t *FullRedrawThrottler) MarkFlushed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastFlushTime = time.Now()
}

// GetNextCheckDelay returns the delay until the next flush should be attempted.
// Use this to schedule a timer when ShouldFlush returns false.
func (t *FullRedrawThrottler) GetNextCheckDelay() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	delay := t.calculateDelayLocked()
	if delay == 0 {
		// Not throttling - use a short default
		return 50 * time.Millisecond
	}

	// Return time remaining until next allowed flush
	elapsed := time.Since(t.lastFlushTime)
	remaining := delay - elapsed
	if remaining < 0 {
		remaining = 0
	}

	// Add small buffer to avoid racing
	return remaining + 10*time.Millisecond
}

// GetCurrentDelay returns the current throttle delay (for debugging/testing).
// Returns 0 if not throttling.
func (t *FullRedrawThrottler) GetCurrentDelay() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calculateDelayLocked()
}

// GetFrequency returns the current redraw frequency (redraws/second) within the effective window.
func (t *FullRedrawThrottler) GetFrequency() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanExpiredLocked(time.Now())
	return t.getFrequencyLocked()
}

// GetBandwidth returns the current bandwidth (bytes/second) within the effective window.
func (t *FullRedrawThrottler) GetBandwidth() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanExpiredLocked(time.Now())
	return t.getBandwidthLocked()
}

// GetEffectiveWindowSize returns the current effective window size.
func (t *FullRedrawThrottler) GetEffectiveWindowSize() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.effectiveWindowSize
}

// IsThrottling returns whether throttling is currently active.
func (t *FullRedrawThrottler) IsThrottling() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cleanExpiredLocked(time.Now())
	return t.calculateDelayLocked() > 0
}

// Reset clears the throttler state, disabling any active throttling.
func (t *FullRedrawThrottler) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.redrawRecords = t.redrawRecords[:0]
	t.lastFlushTime = time.Time{}
	t.effectiveWindowSize = t.baseWindowSize
}

// cleanExpiredLocked removes entries outside the effective sliding window.
// Must be called with lock held.
func (t *FullRedrawThrottler) cleanExpiredLocked(now time.Time) {
	// Use max window for cleanup to preserve data for bandwidth calculation
	// The effective window is used for frequency calculation
	cutoff := now.Add(-t.maxWindowSize)

	// Find first valid entry
	validStart := 0
	for i, rec := range t.redrawRecords {
		if rec.time.After(cutoff) {
			validStart = i
			break
		}
		validStart = i + 1
	}

	if validStart > 0 {
		// Shift valid entries to front
		copy(t.redrawRecords, t.redrawRecords[validStart:])
		t.redrawRecords = t.redrawRecords[:len(t.redrawRecords)-validStart]
	}
}

// adjustWindowLocked adjusts the effective window size based on current bandwidth.
// Higher bandwidth → larger window → more aggressive throttling.
// Must be called with lock held.
func (t *FullRedrawThrottler) adjustWindowLocked() {
	bandwidth := t.getBandwidthLocked()

	if bandwidth <= t.lowBandwidthThreshold {
		// Low bandwidth - use minimum window for quick response
		t.effectiveWindowSize = t.minWindowSize
	} else if bandwidth >= t.highBandwidthThreshold {
		// High bandwidth - use maximum window for aggressive throttling
		t.effectiveWindowSize = t.maxWindowSize
	} else {
		// Linear interpolation between min and max
		ratio := float64(bandwidth-t.lowBandwidthThreshold) /
			float64(t.highBandwidthThreshold-t.lowBandwidthThreshold)
		windowRange := t.maxWindowSize - t.minWindowSize
		t.effectiveWindowSize = t.minWindowSize + time.Duration(float64(windowRange)*ratio)
	}
}

// getFrequencyLocked calculates the current frequency (redraws/second).
// Only counts records within the effective window.
// Must be called with lock held.
func (t *FullRedrawThrottler) getFrequencyLocked() float64 {
	if len(t.redrawRecords) == 0 {
		return 0
	}

	// Count records within effective window
	now := time.Now()
	cutoff := now.Add(-t.effectiveWindowSize)
	count := 0
	for _, rec := range t.redrawRecords {
		if rec.time.After(cutoff) {
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return float64(count) / t.effectiveWindowSize.Seconds()
}

// getBandwidthLocked calculates the current bandwidth (bytes/second).
// Uses the base window size for consistent bandwidth measurement.
// Must be called with lock held.
func (t *FullRedrawThrottler) getBandwidthLocked() int {
	if len(t.redrawRecords) == 0 {
		return 0
	}

	// Sum bytes within base window for bandwidth calculation
	now := time.Now()
	cutoff := now.Add(-t.baseWindowSize)
	totalBytes := 0
	for _, rec := range t.redrawRecords {
		if rec.time.After(cutoff) {
			totalBytes += rec.bytes
		}
	}

	return int(float64(totalBytes) / t.baseWindowSize.Seconds())
}

// calculateDelayLocked calculates the adaptive throttle delay.
// Uses both frequency and bandwidth to determine throttling intensity.
// Must be called with lock held.
//
// Delay calculation:
// - Below threshold: 0 (no delay)
// - At threshold: minDelay (200ms default)
// - Linear increase based on frequency and bandwidth
func (t *FullRedrawThrottler) calculateDelayLocked() time.Duration {
	frequency := t.getFrequencyLocked()
	bandwidth := t.getBandwidthLocked()

	// Check frequency threshold
	if frequency < t.thresholdFreq {
		// Also check bandwidth - high bandwidth alone can trigger throttling
		// If bandwidth > high threshold but frequency is low, still throttle
		if bandwidth < t.highBandwidthThreshold {
			return 0 // Below both thresholds, no throttling
		}
	}

	// Calculate frequency-based delay component
	maxFreq := 10.0
	freqRatio := (frequency - t.thresholdFreq) / (maxFreq - t.thresholdFreq)
	if freqRatio < 0 {
		freqRatio = 0
	}
	if freqRatio > 1 {
		freqRatio = 1
	}

	// Calculate bandwidth-based delay component
	bandwidthRatio := float64(bandwidth-t.lowBandwidthThreshold) /
		float64(t.highBandwidthThreshold-t.lowBandwidthThreshold)
	if bandwidthRatio < 0 {
		bandwidthRatio = 0
	}
	if bandwidthRatio > 1 {
		bandwidthRatio = 1
	}

	// Use the higher of the two ratios (more aggressive throttling)
	ratio := freqRatio
	if bandwidthRatio > ratio {
		ratio = bandwidthRatio
	}

	if ratio == 0 {
		return 0
	}

	delay := t.minDelay + time.Duration(float64(t.maxDelay-t.minDelay)*ratio)
	return delay
}
