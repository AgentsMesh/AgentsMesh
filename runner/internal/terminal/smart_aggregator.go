// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"bytes"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Frame boundary sequences for TUI applications
var (
	// Legacy: ANSI clear screen sequence: ESC[2J
	// Used by traditional terminal apps like `clear` command
	clearScreenSeq = []byte{0x1b, '[', '2', 'J'}

	// Modern: Synchronized Output start sequence: ESC[?2026h
	// Used by Claude Code and modern TUI frameworks (Ink, Bubbletea, etc.)
	// Reference: https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036
	//
	// A complete frame looks like: ESC[?2026h <content> ESC[?2026l
	// When discarding old frames, we find the last frame START (ESC[?2026h)
	// and keep everything from that point onwards.
	// Note: We only need to detect the START sequence for frame boundary detection.
	// The END sequence (ESC[?2026l) is not needed because we keep from START onwards.
	syncOutputStartSeq = []byte{0x1b, '[', '?', '2', '0', '2', '6', 'h'}
)

// SmartAggregator intelligently aggregates TUI output with adaptive frame rate.
//
// Key features:
// - Time-window aggregation (base 50ms = 20 FPS)
// - Frame boundary detection for identifying complete frames:
//   - Primary: Synchronized Output start (ESC[?2026h) - used by Claude Code
//   - Fallback: Clear screen (ESC[2J) - used by traditional apps
// - High load: only keep latest complete frame (discard old frames)
// - Adaptive delay based on queue pressure (50ms → 500ms)
// - Backpressure: pauses when consumer signals overload
// - ttyd-style flow control: propagates backpressure to PTY layer
// - Relay output: optional callback for sending output to Relay in addition to gRPC
// - Serialize mode: use VirtualTerminal.Serialize() for bandwidth optimization
//
// Design principle: TUI apps (like Claude Code) use Synchronized Output mode
// (ESC[?2026h to start, ESC[?2026l to end) for atomic frame updates.
// We detect the START sequence to keep the latest complete frame and discard older ones.
type SmartAggregator struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	timer    *time.Timer
	onFlush  func([]byte)
	stopped  bool

	// Configuration
	baseDelay    time.Duration  // Base delay 16ms (60 FPS)
	maxDelay     time.Duration  // Max delay 200ms (5 FPS)
	maxSize      int            // Max buffer size 64KB
	queueUsageFn func() float64 // Queue usage callback

	// Serialize mode: when set, flush sends VT.Serialize() result instead of raw buffer
	// This enables bandwidth optimization by compressing spaces to CSI CUF sequences
	serializeCallback func() []byte
	hasPendingData    bool // True when there's data to serialize (set by Write, cleared by flush)

	// Backpressure control
	paused    bool          // True when consumer signals overload
	pausedMu  sync.RWMutex  // Protects paused flag
	resumeCh  chan struct{} // Signal from consumer when ready

	// ttyd-style backpressure propagation to PTY layer
	// These callbacks allow backpressure to flow all the way to Terminal.PauseRead/ResumeRead
	onPause  func() // Called when aggregator is paused (propagate to Terminal)
	onResume func() // Called when aggregator is resumed (propagate to Terminal)

	// Relay output callback (for Relay architecture)
	// When set, flushed data is also sent to this callback
	relayOutput   func([]byte)
	relayOutputMu sync.RWMutex
}

// SmartAggregatorOption is a functional option for SmartAggregator.
type SmartAggregatorOption func(*SmartAggregator)

// WithSmartBaseDelay sets the base delay for aggregation.
func WithSmartBaseDelay(d time.Duration) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.baseDelay = d
	}
}

// WithSmartMaxDelay sets the maximum delay for aggregation.
func WithSmartMaxDelay(d time.Duration) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.maxDelay = d
	}
}

// WithSmartMaxSize sets the maximum buffer size.
func WithSmartMaxSize(size int) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.maxSize = size
	}
}

// WithBackpressureCallbacks sets the callbacks for ttyd-style backpressure propagation.
// onPause is called when aggregator is paused (should call Terminal.PauseRead)
// onResume is called when aggregator is resumed (should call Terminal.ResumeRead)
func WithBackpressureCallbacks(onPause, onResume func()) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.onPause = onPause
		a.onResume = onResume
	}
}

// WithSerializeCallback sets a callback that returns serialized terminal content.
// When set, flushLocked() calls this callback to get compressed data instead of using raw buffer.
// This enables bandwidth optimization by using VirtualTerminal.Serialize() which compresses
// spaces to CSI CUF sequences.
//
// In serialize mode:
// - Write() only marks "has pending data", doesn't buffer the actual data
// - flushLocked() calls serializeCallback to get compressed output
// - The callback should return VT.Serialize() result
func WithSerializeCallback(fn func() []byte) SmartAggregatorOption {
	return func(a *SmartAggregator) {
		a.serializeCallback = fn
	}
}

// NewSmartAggregator creates a new smart aggregator.
//
// Parameters:
// - onFlush: callback invoked with aggregated data
// - queueUsageFn: returns queue usage ratio (0.0 to 1.0), used for adaptive delay
func NewSmartAggregator(onFlush func([]byte), queueUsageFn func() float64, opts ...SmartAggregatorOption) *SmartAggregator {
	a := &SmartAggregator{
		onFlush:      onFlush,
		queueUsageFn: queueUsageFn,
		baseDelay:    50 * time.Millisecond,  // 20 FPS (was 60 FPS) - more aggressive aggregation
		maxDelay:     500 * time.Millisecond, // 2 FPS (was 5 FPS) - allow more buffering under load
		maxSize:      8 * 1024,               // 8KB (was 64KB) - smaller chunks, more responsive
		resumeCh:     make(chan struct{}, 1), // Buffered to avoid blocking
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Pause signals the aggregator to pause flushing (called by consumer when overloaded).
// The aggregator will continue buffering data but won't flush until Resume is called.
// Also propagates backpressure to the PTY layer via onPause callback (ttyd-style).
func (a *SmartAggregator) Pause() {
	a.pausedMu.Lock()
	wasPaused := a.paused
	a.paused = true
	a.pausedMu.Unlock()

	// Propagate backpressure to PTY layer (ttyd-style flow control)
	// This stops Terminal from reading more data, preventing memory growth
	if !wasPaused && a.onPause != nil {
		a.onPause()
	}
}

// Resume signals the aggregator to resume flushing (called by consumer when ready).
// This triggers an immediate flush attempt if there's buffered data.
// Also releases backpressure on the PTY layer via onResume callback (ttyd-style).
func (a *SmartAggregator) Resume() {
	a.pausedMu.Lock()
	wasPaused := a.paused
	a.paused = false
	a.pausedMu.Unlock()

	// Signal resume and trigger flush if was paused
	if wasPaused {
		// Propagate resume to PTY layer (ttyd-style flow control)
		// This allows Terminal to resume reading
		if a.onResume != nil {
			a.onResume()
		}

		select {
		case a.resumeCh <- struct{}{}:
		default:
		}
		// Trigger immediate flush check
		go a.timerFlush()
	}
}

// IsPaused returns whether the aggregator is currently paused.
func (a *SmartAggregator) IsPaused() bool {
	a.pausedMu.RLock()
	defer a.pausedMu.RUnlock()
	return a.paused
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

	// Get current queue usage
	usage := 0.0
	if a.queueUsageFn != nil {
		usage = a.queueUsageFn()
	}

	// Check backpressure state
	a.pausedMu.RLock()
	paused := a.paused
	a.pausedMu.RUnlock()

	// Serialize mode: just mark pending data, don't buffer
	if a.serializeCallback != nil {
		a.hasPendingData = true

		logger.Terminal().Debug("SmartAggregator Write (serialize mode)",
			"usage", usage, "paused", paused, "has_timer", a.timer != nil)

		// Critical load (>50%): skip immediate flush, just accumulate
		if usage > 0.5 {
			if a.timer == nil {
				a.timer = time.AfterFunc(a.maxDelay, a.timerFlush)
			}
			return
		}

		// Calculate adaptive delay and schedule flush timer
		delay := a.calculateDelay(usage)
		if a.timer == nil {
			a.timer = time.AfterFunc(delay, a.timerFlush)
		}
		return
	}

	// Legacy mode: buffer raw data
	// Enforce buffer size limit to prevent unbounded memory growth
	// This is critical for cases without clear screen signals (e.g., `find /`)
	a.enforceBufferLimit(len(data))

	a.buffer.Write(data)

	logger.Terminal().Debug("SmartAggregator Write (legacy mode)",
		"data_len", len(data), "buffer_len", a.buffer.Len(),
		"usage", usage, "paused", paused, "has_timer", a.timer != nil)

	// Always discard old frames - TUI apps redraw full screen, old frames are useless
	// This is the most effective way to reduce bandwidth for Claude Code / TUI apps
	a.discardOldFrames()

	// Critical load (>50%): skip immediate flush, just accumulate
	// This prevents flooding the queue when it's already overwhelmed
	if usage > 0.5 {
		// Schedule a recovery check timer if not already scheduled
		if a.timer == nil {
			a.timer = time.AfterFunc(a.maxDelay, a.timerFlush)
		}
		return
	}

	// Calculate adaptive delay based on queue pressure
	delay := a.calculateDelay(usage)

	// Schedule flush timer
	if a.timer == nil {
		a.timer = time.AfterFunc(delay, a.timerFlush)
	}

	// Flush immediately if buffer exceeds max size (but respect critical load)
	if a.buffer.Len() >= a.maxSize && usage < 0.8 {
		a.flushLocked()
	}
}

// enforceBufferLimit ensures buffer doesn't exceed maxSize.
// If adding newDataLen would exceed limit, discards oldest data.
// Must be called with lock held.
func (a *SmartAggregator) enforceBufferLimit(newDataLen int) {
	targetLen := a.buffer.Len() + newDataLen
	if targetLen <= a.maxSize {
		return // Within limit
	}

	// First try to discard old frames (keeps data after last clear screen)
	a.discardOldFrames()

	// Check again after discarding frames
	targetLen = a.buffer.Len() + newDataLen
	if targetLen <= a.maxSize {
		return // Now within limit
	}

	// Still over limit - discard oldest data from head
	// This handles cases without clear screen signals (e.g., `find /` output)
	excess := targetLen - a.maxSize
	if excess > 0 && excess < a.buffer.Len() {
		// Keep only the tail (newest data)
		data := a.buffer.Bytes()
		// Adjust offset to UTF-8 character boundary to avoid breaking multi-byte characters
		offset := alignToUTF8Boundary(data, excess)
		newData := make([]byte, len(data)-offset)
		copy(newData, data[offset:])
		a.buffer.Reset()
		a.buffer.Write(newData)
	} else if excess >= a.buffer.Len() {
		// New data alone exceeds limit - clear buffer entirely
		// (new data will be truncated by caller if needed)
		a.buffer.Reset()
	}
}

// alignToUTF8Boundary adjusts an offset to the next valid UTF-8 character boundary.
// This prevents truncating in the middle of a multi-byte UTF-8 character.
// If offset is already at a boundary, it returns offset unchanged.
// If offset is in the middle of a multi-byte sequence, it advances to the next valid start.
func alignToUTF8Boundary(data []byte, offset int) int {
	if offset >= len(data) {
		return len(data)
	}
	// If we're at the start of a valid UTF-8 character, we're done
	if utf8.RuneStart(data[offset]) {
		return offset
	}
	// Otherwise, advance until we find the start of a valid UTF-8 character
	// UTF-8 continuation bytes have the form 10xxxxxx (0x80-0xBF)
	// Valid start bytes have other forms
	for offset < len(data) && !utf8.RuneStart(data[offset]) {
		offset++
	}
	return offset
}

// findLastValidUTF8Boundary finds the last position in data that ends on a valid UTF-8 boundary.
// This is used to avoid sending incomplete multi-byte characters at the end of a message.
// Returns the length of the valid portion (may be equal to len(data) if data ends on a boundary).
func findLastValidUTF8Boundary(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Check if data already ends on a valid UTF-8 boundary
	// by scanning backwards from the end to find the start of the last character
	for i := len(data) - 1; i >= 0 && i >= len(data)-4; i-- {
		if utf8.RuneStart(data[i]) {
			// Found the start of a UTF-8 character
			// Check if the remaining bytes form a complete character
			r, size := utf8.DecodeRune(data[i:])
			if r != utf8.RuneError || size == len(data)-i {
				// Complete character or valid single byte
				return len(data)
			}
			// Incomplete character - truncate before it
			return i
		}
	}

	// All bytes in the last 4 positions are continuation bytes
	// This shouldn't happen with valid UTF-8, but handle it anyway
	// by finding where the valid data ends
	for i := len(data) - 1; i >= 0; i-- {
		if utf8.RuneStart(data[i]) {
			return i
		}
	}

	// No valid UTF-8 start byte found - data might be binary or corrupted
	// Return all data to avoid infinite buffering
	return len(data)
}

// discardOldFrames keeps only the latest complete frame.
// This ensures we send complete frames, avoiding screen tearing.
//
// Frame boundary detection priority:
// 1. Synchronized Output start (ESC[?2026h) - used by Claude Code and modern TUI apps
//    A frame is: ESC[?2026h <content> ESC[?2026l
//    We keep from the LAST frame start onwards to preserve the newest frame
// 2. Clear screen (ESC[2J) - used by traditional terminal apps
func (a *SmartAggregator) discardOldFrames() {
	data := a.buffer.Bytes()

	// Priority 1: Check for Synchronized Output START sequence
	// Claude Code uses ESC[?2026h to begin a frame and ESC[?2026l to end it.
	// We find the last frame start to keep the newest complete frame.
	if idx := bytes.LastIndex(data, syncOutputStartSeq); idx > 0 {
		// Keep only content from the last frame start onwards
		discardLen := idx
		newData := make([]byte, len(data)-idx)
		copy(newData, data[idx:])
		a.buffer.Reset()
		a.buffer.Write(newData)
		logger.Terminal().Debug("SmartAggregator discarded old frames (sync output start)",
			"discarded_bytes", discardLen, "kept_bytes", len(newData))
		return
	}

	// Priority 2: Fallback to clear screen sequence (legacy apps)
	if idx := bytes.LastIndex(data, clearScreenSeq); idx > 0 {
		// Keep only content from the last clear screen onwards
		discardLen := idx
		newData := make([]byte, len(data)-idx)
		copy(newData, data[idx:])
		a.buffer.Reset()
		a.buffer.Write(newData)
		logger.Terminal().Debug("SmartAggregator discarded old frames (clear screen)",
			"discarded_bytes", discardLen, "kept_bytes", len(newData))
	}
}

// calculateDelay computes adaptive delay based on queue usage.
//
// Load mapping:
// - 0%   → 16ms  (60 FPS) - smooth performance
// - 50%  → 50ms  (20 FPS) - moderate throttling
// - 80%  → 100ms (10 FPS) - significant throttling
// - 100% → 200ms (5 FPS)  - maximum throttling
func (a *SmartAggregator) calculateDelay(usage float64) time.Duration {
	// Quadratic scaling: factor = 1 + usage² * 12
	// This gives smooth ramp-up with aggressive throttling at high load
	factor := 1.0 + (usage * usage * 12)
	delay := time.Duration(float64(a.baseDelay) * factor)
	if delay > a.maxDelay {
		return a.maxDelay
	}
	return delay
}

// timerFlush is called when the timer fires.
func (a *SmartAggregator) timerFlush() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopped {
		return
	}

	// Check if paused by consumer (backpressure)
	a.pausedMu.RLock()
	paused := a.paused
	a.pausedMu.RUnlock()

	if paused {
		// Still paused, reschedule check
		logger.Terminal().Debug("SmartAggregator timerFlush: paused, rescheduling",
			"buffer_len", a.buffer.Len())
		a.discardOldFrames()
		a.timer = time.AfterFunc(a.maxDelay, a.timerFlush)
		return
	}

	// Check queue usage before flushing
	usage := 0.0
	if a.queueUsageFn != nil {
		usage = a.queueUsageFn()
	}

	// If still in critical load, reschedule instead of flushing
	if usage > 0.5 {
		logger.Terminal().Debug("SmartAggregator timerFlush: critical load, rescheduling",
			"usage", usage, "buffer_len", a.buffer.Len())
		a.discardOldFrames()
		// Reschedule with longer delay to check again
		a.timer = time.AfterFunc(a.maxDelay, a.timerFlush)
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

		logger.Terminal().Debug("SmartAggregator flushing (serialize mode)", "bytes", len(data))
	} else {
		// Legacy mode: use raw buffer data

		if a.buffer.Len() == 0 {
			return
		}

		// Get all buffered data
		allData := a.buffer.Bytes()

		// Find the last valid UTF-8 boundary to avoid sending incomplete multi-byte characters
		// This prevents garbled text when data is split across messages
		validLen := findLastValidUTF8Boundary(allData)

		// Copy valid data for sending
		data = make([]byte, validLen)
		copy(data, allData[:validLen])

		// Keep any trailing incomplete UTF-8 bytes in the buffer for next flush
		if validLen < len(allData) {
			remaining := make([]byte, len(allData)-validLen)
			copy(remaining, allData[validLen:])
			a.buffer.Reset()
			a.buffer.Write(remaining)
			logger.Terminal().Debug("SmartAggregator keeping incomplete UTF-8", "remaining", len(remaining))
		} else {
			a.buffer.Reset()
		}

		if len(data) == 0 {
			return
		}

		// NOTE: Space compression (compressSpaces) was attempted but DOES NOT WORK for TUI apps.
		// Problem: CSI CUF (\x1b[nC) only moves cursor, it does NOT overwrite existing content.
		// When TUI apps redraw the screen, they rely on spaces to clear old content.
		// Using CUF instead of spaces leaves old content (like box-drawing chars) visible.
		// The correct solution requires a full VirtualTerminal implementation, not simple compression.
		// Now we use serialize mode with VirtualTerminal.Serialize() for proper space compression.

		logger.Terminal().Debug("SmartAggregator flushing (legacy mode)", "bytes", len(data))
	}

	// Get relay output callback (if set)
	relayOutput := a.getRelayOutput()

	// Call callbacks outside lock to avoid deadlock
	// Use goroutine to avoid blocking the lock
	go func() {
		// Priority: Relay mode > Legacy gRPC mode
		// When Relay is connected, send data ONLY through Relay (not gRPC)
		// This avoids duplicate data transmission and reduces Backend load
		if relayOutput != nil {
			// Relay mode: send through Relay WebSocket
			logger.Terminal().Debug("SmartAggregator sending to relay", "bytes", len(data))
			relayOutput(data)
		} else if a.onFlush != nil {
			// Legacy fallback: send through gRPC when no Relay connected
			// This ensures backward compatibility during migration
			logger.Terminal().Debug("SmartAggregator sending to gRPC (no relay)", "bytes", len(data))
			a.onFlush(data)
		} else {
			logger.Terminal().Warn("SmartAggregator: no output callback set", "bytes", len(data))
		}
	}()
}

// Flush forces an immediate flush of the buffer.
// Thread-safe: can be called from any goroutine.
func (a *SmartAggregator) Flush() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopped {
		a.flushLocked()
	}
}

// Stop stops the aggregator and flushes remaining data.
// After Stop(), Write() calls are ignored.
func (a *SmartAggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopped {
		return
	}
	a.stopped = true
	a.flushLocked()
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
// When set, flushed data is also sent to this callback in addition to the main onFlush callback.
// Pass nil to disable relay output.
// Thread-safe: can be called from any goroutine.
func (a *SmartAggregator) SetRelayOutput(fn func([]byte)) {
	a.relayOutputMu.Lock()
	defer a.relayOutputMu.Unlock()
	a.relayOutput = fn
}

// getRelayOutput returns the relay output callback.
func (a *SmartAggregator) getRelayOutput() func([]byte) {
	a.relayOutputMu.RLock()
	defer a.relayOutputMu.RUnlock()
	return a.relayOutput
}
