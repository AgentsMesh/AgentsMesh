// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// SmartAggregator intelligently aggregates TUI output with adaptive frame rate.
//
// Key features:
// - Time-window aggregation (base 50ms = 20 FPS)
// - Frame boundary detection with complete frame preservation:
//   - Primary: Synchronized Output (ESC[?2026h / ESC[?2026l) - used by Claude Code
//   - Fallback: Clear screen (ESC[2J) - used by traditional apps
// - Frame-aware flushing: incomplete frames are kept in buffer until complete
// - Adaptive delay based on queue pressure (50ms → 500ms)
// - Backpressure: pauses when consumer signals overload
// - ttyd-style flow control: propagates backpressure to PTY layer
// - Relay output: optional callback for sending output to Relay in addition to gRPC
// - Serialize mode: use VirtualTerminal.Serialize() for bandwidth optimization
// - Full redraw throttling: detects high-frequency redraws and reduces transmission rate
//
// Design principle: TUI apps (like Claude Code) use Synchronized Output mode
// (ESC[?2026h to start, ESC[?2026l to end) for atomic frame updates.
// We preserve complete frames and don't flush incomplete frames to avoid screen tearing.
type SmartAggregator struct {
	mu      sync.Mutex
	stopped bool
	timer   *time.Timer

	// Composed components (SRP: each handles one responsibility)
	buffer       *FrameBuffer
	delay        *AdaptiveDelay
	backpressure *BackpressureController
	router       *OutputRouter

	// Serialize mode: when set, flush sends VT.Serialize() result instead of raw buffer
	// This enables bandwidth optimization by compressing spaces to CSI CUF sequences
	serializeCallback func() []byte
	hasPendingData    bool // True when there's data to serialize (set by Write, cleared by flush)

	// Full redraw throttling (Legacy mode only, i.e., non-Serialize mode)
	// Detects high-frequency full-screen redraws and reduces transmission rate
	fullRedrawThrottler *FullRedrawThrottler

	// PTY logging (for debugging)
	ptyLogger *PTYLogger
}

// Note: SmartAggregatorOption and With* functions are in smart_aggregator_options.go

// NewSmartAggregator creates a new smart aggregator.
//
// Parameters:
// - onFlush: callback invoked with aggregated data
// - queueUsageFn: returns queue usage ratio (0.0 to 1.0), used for adaptive delay
func NewSmartAggregator(onFlush func([]byte), queueUsageFn func() float64, opts ...SmartAggregatorOption) *SmartAggregator {
	// Default configuration
	baseDelay := 50 * time.Millisecond  // 20 FPS - more aggressive aggregation
	maxDelay := 500 * time.Millisecond  // 2 FPS - allow more buffering under load
	maxSize := 1024 * 1024              // 1MB - generous buffer to avoid any truncation issues

	a := &SmartAggregator{
		buffer:       NewFrameBuffer(maxSize),
		delay:        NewAdaptiveDelay(baseDelay, maxDelay, queueUsageFn),
		backpressure: NewBackpressureController(nil, nil),
		router:       NewOutputRouter(onFlush),
	}

	for _, opt := range opts {
		opt(a)
	}

	logger.Terminal().Debug("SmartAggregator created",
		"base_delay", baseDelay,
		"max_delay", maxDelay,
		"max_size", maxSize)

	return a
}

// Pause signals the aggregator to pause flushing (called by consumer when overloaded).
// The aggregator will continue buffering data but won't flush until Resume is called.
// Also propagates backpressure to the PTY layer via onPause callback (ttyd-style).
func (a *SmartAggregator) Pause() {
	logger.Terminal().Debug("SmartAggregator pausing")
	a.backpressure.Pause()
}

// Resume signals the aggregator to resume flushing (called by consumer when ready).
// This triggers an immediate flush attempt if there's buffered data.
// Also releases backpressure on the PTY layer via onResume callback (ttyd-style).
func (a *SmartAggregator) Resume() {
	logger.Terminal().Debug("SmartAggregator resuming")
	if a.backpressure.Resume() {
		// Trigger immediate flush check
		go a.timerFlush()
	}
}

// IsPaused returns whether the aggregator is currently paused.
func (a *SmartAggregator) IsPaused() bool {
	return a.backpressure.IsPaused()
}

// Write adds data to the aggregation buffer.
// Thread-safe: can be called from multiple goroutines.
// Buffer is hard-capped at maxSize to prevent unbounded memory growth.
//
// In serialize mode (when serializeCallback is set):
// - The data parameter is ignored (can be nil)
// - Only marks that there's pending data to serialize
// - Actual data comes from serializeCallback during flush
func (a *SmartAggregator) Write(data []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stopped {
		return
	}

	// Log raw PTY output if logger is set
	if a.ptyLogger != nil && len(data) > 0 {
		a.ptyLogger.WriteRaw(data)
	}

	usage := a.delay.GetUsage()
	paused := a.backpressure.IsPaused()

	// Serialize mode: just mark pending data, don't buffer
	if a.serializeCallback != nil {
		a.hasPendingData = true

		logger.TerminalTrace().Trace("SmartAggregator Write (serialize mode)",
			"usage", usage, "paused", paused, "has_timer", a.timer != nil)

		// Critical load (>50%): skip immediate flush, just accumulate
		if a.delay.IsCriticalLoad() {
			if a.timer == nil {
				a.timer = time.AfterFunc(a.delay.MaxDelay(), a.timerFlush)
			}
			return
		}

		// Calculate adaptive delay and schedule flush timer
		delay := a.delay.Calculate()
		if a.timer == nil {
			a.timer = time.AfterFunc(delay, a.timerFlush)
		}
		return
	}

	// Legacy mode: buffer raw data with frame-aware management
	a.buffer.Write(data)

	logger.TerminalTrace().Trace("SmartAggregator Write (legacy mode)",
		"data_len", len(data), "buffer_len", a.buffer.Len(),
		"usage", usage, "paused", paused, "has_timer", a.timer != nil)

	// Critical load (>50%): skip immediate flush, just accumulate
	if a.delay.IsCriticalLoad() {
		if a.timer == nil {
			a.timer = time.AfterFunc(a.delay.MaxDelay(), a.timerFlush)
		}
		return
	}

	// Calculate adaptive delay based on queue pressure
	delay := a.delay.Calculate()

	// Schedule flush timer
	if a.timer == nil {
		a.timer = time.AfterFunc(delay, a.timerFlush)
	}

	// Flush immediately if buffer exceeds max size (but respect high load)
	if a.buffer.Len() >= a.buffer.MaxSize() && !a.delay.IsHighLoad() {
		a.flushLocked()
	}
}

// Note: timerFlush and flushLocked are in smart_aggregator_flush.go

// Flush forces an immediate flush of the buffer.
// Thread-safe: can be called from any goroutine.
func (a *SmartAggregator) Flush() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopped {
		a.forceFlushLocked()
	}
}

// Note: forceFlushLocked is in smart_aggregator_flush.go

// Stop stops the aggregator and flushes remaining data.
// After Stop(), Write() calls are ignored.
func (a *SmartAggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopped {
		return
	}
	a.stopped = true
	logger.Terminal().Info("SmartAggregator stopped")
	a.forceFlushLocked()
}

// IsStopped returns whether the aggregator has been stopped.
func (a *SmartAggregator) IsStopped() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stopped
}

// BufferLen returns the current buffer length (for testing/debugging).
func (a *SmartAggregator) BufferLen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.Len()
}

// SetRelayOutput sets the relay output callback.
// When set, flushed data is sent through Relay instead of gRPC.
// Pass nil to disable relay output.
// Thread-safe: can be called from any goroutine.
func (a *SmartAggregator) SetRelayOutput(fn func([]byte)) {
	a.router.SetRelayOutput(fn)
}

// SetPTYLogger sets the PTY logger for debugging.
// When set, raw input and aggregated output are logged to files.
func (a *SmartAggregator) SetPTYLogger(logger *PTYLogger) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ptyLogger = logger
}

// calculateDelay is kept for backward compatibility with tests.
// Delegates to AdaptiveDelay component.
func (a *SmartAggregator) calculateDelay(usage float64) time.Duration {
	return a.delay.CalculateForUsage(usage)
}
