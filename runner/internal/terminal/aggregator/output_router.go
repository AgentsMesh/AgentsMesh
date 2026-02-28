// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// earlyBufferMaxSize is the maximum size for buffering output before any callback is set.
// 64KB is generous enough for error messages and early startup output.
const earlyBufferMaxSize = 64 * 1024

// OutputRouter routes terminal output to appropriate destinations.
// Supports both legacy gRPC output and modern Relay WebSocket output.
//
// Priority: Relay > gRPC (when Relay is connected, gRPC is not used)
//
// Early buffer: when no callback is set (during the window between pod creation
// and relay subscription), output is buffered internally. The buffer is replayed
// when a callback is first set, ensuring no output is lost during startup.
type OutputRouter struct {
	mu          sync.RWMutex
	onFlush     func([]byte) // gRPC fallback callback
	relayOutput func([]byte) // Relay preferred callback

	// Early buffer: captures output before any callback is set.
	// Once a callback is set and the buffer is drained, no further buffering occurs.
	earlyBuffer []byte
	earlyDone   bool // true after buffer has been drained (prevents re-buffering)
}

// NewOutputRouter creates a new output router.
//
// Parameters:
// - onFlush: legacy gRPC callback (always set)
func NewOutputRouter(onFlush func([]byte)) *OutputRouter {
	return &OutputRouter{
		onFlush: onFlush,
	}
}

// Route sends data to the appropriate destination.
// Priority: Relay > gRPC > early buffer
//
// This method is safe to call from any goroutine.
func (r *OutputRouter) Route(data []byte) {
	if len(data) == 0 {
		return
	}

	// Fast path: read callbacks under RLock
	r.mu.RLock()
	relayOutput := r.relayOutput
	onFlush := r.onFlush
	r.mu.RUnlock()

	log := logger.TerminalTrace()

	// Priority: Relay mode > Legacy gRPC mode
	if relayOutput != nil {
		log.Trace("OutputRouter: sending to relay", "bytes", len(data))
		relayOutput(data)
		return
	}
	if onFlush != nil {
		log.Trace("OutputRouter: sending to gRPC (no relay)", "bytes", len(data))
		onFlush(data)
		return
	}

	// No callback set — buffer for later replay
	r.mu.Lock()
	// Re-check under write lock (callback may have been set between RUnlock and Lock)
	if r.relayOutput != nil {
		fn := r.relayOutput
		r.mu.Unlock()
		fn(data)
		return
	}
	if r.onFlush != nil {
		fn := r.onFlush
		r.mu.Unlock()
		fn(data)
		return
	}
	if !r.earlyDone {
		remaining := earlyBufferMaxSize - len(r.earlyBuffer)
		if remaining > 0 {
			if len(data) > remaining {
				data = data[:remaining]
			}
			r.earlyBuffer = append(r.earlyBuffer, data...)
			log.Debug("OutputRouter: buffered early output", "bytes", len(data), "total_buffered", len(r.earlyBuffer))
		} else {
			log.Warn("OutputRouter: early buffer full, dropping output", "bytes", len(data))
		}
	}
	r.mu.Unlock()
}

// SetRelayOutput sets the relay output callback.
// If there is buffered early output, it is replayed through the new callback immediately.
// Pass nil to disable relay output and fall back to gRPC.
// Thread-safe.
func (r *OutputRouter) SetRelayOutput(fn func([]byte)) {
	r.mu.Lock()
	r.relayOutput = fn
	buffered := r.drainEarlyBufferLocked()
	r.mu.Unlock()

	// Replay buffered data outside the lock to avoid deadlocks
	if fn != nil && len(buffered) > 0 {
		logger.Terminal().Info("OutputRouter: replaying early buffer via relay", "bytes", len(buffered))
		fn(buffered)
	}
}

// GetRelayOutput returns the current relay output callback.
func (r *OutputRouter) GetRelayOutput() func([]byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relayOutput
}

// HasRelayOutput returns whether relay output is configured.
func (r *OutputRouter) HasRelayOutput() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relayOutput != nil
}

// SetOnFlush updates the gRPC callback.
// If there is buffered early output, it is replayed through the new callback immediately.
func (r *OutputRouter) SetOnFlush(fn func([]byte)) {
	r.mu.Lock()
	r.onFlush = fn
	buffered := r.drainEarlyBufferLocked()
	r.mu.Unlock()

	// Replay buffered data outside the lock
	if fn != nil && len(buffered) > 0 {
		logger.Terminal().Info("OutputRouter: replaying early buffer via gRPC", "bytes", len(buffered))
		fn(buffered)
	}
}

// DrainEarlyBuffer returns and clears any buffered early output.
// This is used by the exit handler to retrieve output that was never sent
// (e.g., when the process exits before the relay connects).
// After calling this, further output will not be buffered.
func (r *OutputRouter) DrainEarlyBuffer() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.drainEarlyBufferLocked()
}

// drainEarlyBufferLocked returns and clears the early buffer. Must be called with lock held.
func (r *OutputRouter) drainEarlyBufferLocked() []byte {
	if len(r.earlyBuffer) == 0 {
		r.earlyDone = true
		return nil
	}
	buf := r.earlyBuffer
	r.earlyBuffer = nil
	r.earlyDone = true
	return buf
}
