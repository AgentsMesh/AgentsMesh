// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

const defaultOutputDrainTimeout = 2 * time.Second

// SmartAggregator preserves raw ANSI order while batching output. Synchronized
// output sequences define atomic flush boundaries; they do not replace terminal
// state. Each destination is delivered through its own bounded recovery lane.
type SmartAggregator struct {
	mu           sync.Mutex
	stopped      bool
	timer        *time.Timer
	stopDone     chan struct{}
	drainTimeout time.Duration

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
// - queueUsageFn: returns queue usage ratio (0.0 to 1.0), used for adaptive delay
func NewSmartAggregator(queueUsageFn func() float64, opts ...SmartAggregatorOption) *SmartAggregator {
	// Default configuration
	baseDelay := 50 * time.Millisecond // 20 FPS - more aggressive aggregation
	maxDelay := 500 * time.Millisecond // 2 FPS - allow more buffering under load
	maxSize := 1024 * 1024             // 1MB - generous buffer to avoid any truncation issues

	router := NewOutputRouter()
	if queueUsageFn == nil {
		// Destination queues are isolated failure domains. Feeding their maximum
		// back into the shared aggregation timer would let one slow sink delay every
		// healthy sink; bounded lanes recover independently through snapshots.
		queueUsageFn = func() float64 { return 0 }
	}
	a := &SmartAggregator{
		buffer:       NewFrameBuffer(maxSize),
		delay:        NewAdaptiveDelay(baseDelay, maxDelay, queueUsageFn),
		backpressure: NewBackpressureController(nil, nil),
		router:       router,
		stopDone:     make(chan struct{}),
		drainTimeout: defaultOutputDrainTimeout,
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
// Explicit callbacks may propagate this state to an upstream producer.
func (a *SmartAggregator) Pause() {
	logger.Terminal().Debug("SmartAggregator pausing")
	a.backpressure.Pause()
}

// Resume signals the aggregator to resume flushing (called by consumer when ready).
// This triggers an immediate flush attempt if there's buffered data.
// Explicit callbacks may propagate this state to an upstream producer.
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
// Bulk buffering is capped at maxSize; UTF-8 completion has fixed three-byte
// boundary headroom so overflow recovery cannot split continuation ownership.
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
	if !a.buffer.Write(data) {
		logger.Terminal().Warn("SmartAggregator raw buffer overflow; scheduling snapshot recovery",
			"data_len", len(data), "max_size", a.buffer.MaxSize())
		a.router.desynchronizeAll(OutputDesyncBuffer, nil)
		return
	}

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

// Flush and lifecycle methods are in smart_aggregator_lifecycle.go.
