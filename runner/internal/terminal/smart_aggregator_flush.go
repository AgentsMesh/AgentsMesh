// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// timerFlush is called when the timer fires.
func (a *SmartAggregator) timerFlush() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.stopped {
		return
	}

	log := logger.TerminalTrace()

	// Check if paused by consumer (backpressure)
	if a.backpressure.IsPaused() {
		log.Trace("SmartAggregator timerFlush: paused, rescheduling",
			"buffer_len", a.buffer.Len())
		a.timer = time.AfterFunc(a.delay.MaxDelay(), a.timerFlush)
		return
	}

	// If still in critical load, reschedule instead of flushing
	if a.delay.IsCriticalLoad() {
		log.Trace("SmartAggregator timerFlush: critical load, rescheduling",
			"usage", a.delay.GetUsage(), "buffer_len", a.buffer.Len())
		a.timer = time.AfterFunc(a.delay.MaxDelay(), a.timerFlush)
		return
	}

	a.flushLocked()
}

// flushLocked flushes the buffer. Must be called with lock held.
func (a *SmartAggregator) flushLocked() {
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}

	var data []byte

	// Serialize mode: use callback to get compressed data from VirtualTerminal
	if a.serializeCallback != nil {
		// Check if there's pending data to serialize
		if !a.hasPendingData {
			return
		}

		// Get serialized data from VirtualTerminal
		data = a.serializeCallback()
		a.hasPendingData = false
		a.buffer.Reset() // Clear any trigger markers

		if len(data) == 0 {
			return
		}

		logger.TerminalTrace().Trace("SmartAggregator flushing (serialize mode)", "bytes", len(data))
	} else {
		// Legacy mode: use frame-aware buffer flush
		// FlushComplete ensures we don't break incomplete frames

		// Full redraw throttling: detect high-frequency redraws and reduce transmission rate
		if a.fullRedrawThrottler != nil && a.buffer.IsLastFrameFullRedraw() {
			// Record redraw with frame size for bandwidth-aware throttling
			a.fullRedrawThrottler.RecordRedraw(a.buffer.Len())

			// Check if we should throttle (skip this flush)
			if !a.fullRedrawThrottler.ShouldFlush() {
				// Throttling: skip this flush, schedule next check
				// Data stays in buffer until next allowed flush
				delay := a.fullRedrawThrottler.GetNextCheckDelay()
				a.timer = time.AfterFunc(delay, a.timerFlush)
				logger.TerminalTrace().Trace("SmartAggregator: throttling full redraw",
					"next_check", delay,
					"frequency", a.fullRedrawThrottler.GetFrequency(),
					"bandwidth_kbps", a.fullRedrawThrottler.GetBandwidth()/1024,
					"effective_window", a.fullRedrawThrottler.GetEffectiveWindowSize())
				return
			}
		}

		var remaining int
		data, remaining = a.buffer.FlushComplete()

		if len(data) == 0 {
			// No complete frames to flush - reschedule if there's data
			if remaining > 0 {
				// There's an incomplete frame - schedule check for when it completes
				a.timer = time.AfterFunc(a.delay.Calculate(), a.timerFlush)
			}
			return
		}

		logger.TerminalTrace().Trace("SmartAggregator flushing (legacy mode)",
			"bytes", len(data), "remaining", remaining)
	}

	// Mark flush time for throttler (if active)
	if a.fullRedrawThrottler != nil {
		a.fullRedrawThrottler.MarkFlushed()
	}

	// Log aggregated output if logger is set
	if a.ptyLogger != nil && len(data) > 0 {
		a.ptyLogger.WriteAggregated(data)
	}

	// Route output (async to avoid holding lock)
	dataCopy := data
	go a.router.Route(dataCopy)
}

// forceFlushLocked flushes all data including incomplete frames.
// Used by Flush() and Stop().
func (a *SmartAggregator) forceFlushLocked() {
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}

	var data []byte

	if a.serializeCallback != nil {
		if !a.hasPendingData {
			return
		}
		data = a.serializeCallback()
		a.hasPendingData = false
		a.buffer.Reset()
	} else {
		// FlushAll flushes everything including incomplete frames
		var _ int
		data, _ = a.buffer.FlushAll()
	}

	if len(data) == 0 {
		return
	}

	// Log aggregated output if logger is set
	if a.ptyLogger != nil {
		a.ptyLogger.WriteAggregated(data)
	}

	logger.TerminalTrace().Trace("SmartAggregator force flushing", "bytes", len(data))

	// Route output (async to avoid holding lock)
	dataCopy := data
	go a.router.Route(dataCopy)
}
