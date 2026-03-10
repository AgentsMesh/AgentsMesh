// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// earlyBufferMaxSize is the maximum size for buffering output before any callback is set.
// 64KB is generous enough for error messages and early startup output.
const earlyBufferMaxSize = 64 * 1024

// RelayWriter abstracts the relay client for output routing.
// Route() checks IsConnected() at call time, eliminating stale-closure races.
type RelayWriter interface {
	SendOutput(data []byte) error
	IsConnected() bool
}

// OutputRouter routes terminal output to appropriate destinations.
// Supports both legacy gRPC output and modern Relay WebSocket output.
//
// Priority: Relay > gRPC (when Relay is connected, gRPC is not used)
// When Relay is registered but disconnected, output falls back to gRPC automatically.
//
// Early buffer: when no callback is set (during the window between pod creation
// and relay subscription), output is buffered internally. The buffer is replayed
// when a callback is first set, ensuring no output is lost during startup.
type OutputRouter struct {
	mu      sync.RWMutex
	onFlush func([]byte)  // gRPC fallback callback
	relay   RelayWriter   // Relay client reference (checked at Route time)

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
// Priority: Relay (connected) > gRPC > early buffer
//
// Unlike callback-based routing, this checks relay.IsConnected() at call time,
// so stale closures from old relay clients cannot intercept output.
// When relay is registered but disconnected (e.g., during reconnect), output
// automatically falls back to gRPC — no silent data loss.
//
// This method is safe to call from any goroutine.
func (r *OutputRouter) Route(data []byte) {
	if len(data) == 0 {
		return
	}

	// Fast path: read fields under RLock
	r.mu.RLock()
	relay := r.relay
	onFlush := r.onFlush
	r.mu.RUnlock()

	log := logger.TerminalTrace()

	// Priority: Relay mode (only when connected) > Legacy gRPC mode
	if relay != nil && relay.IsConnected() {
		log.Trace("OutputRouter: sending to relay", "bytes", len(data))
		if err := relay.SendOutput(data); err != nil {
			log.Trace("OutputRouter: relay send failed, falling back to gRPC", "bytes", len(data), "error", err)
			if onFlush != nil {
				onFlush(data)
			}
		}
		return
	}
	if onFlush != nil {
		if relay != nil {
			log.Trace("OutputRouter: relay disconnected, falling back to gRPC", "bytes", len(data))
		} else {
			log.Trace("OutputRouter: sending to gRPC (no relay)", "bytes", len(data))
		}
		onFlush(data)
		return
	}

	// No callback set — buffer for later replay
	r.mu.Lock()
	// Re-check under write lock (fields may have been set between RUnlock and Lock)
	if r.relay != nil && r.relay.IsConnected() {
		rl := r.relay
		r.mu.Unlock()
		if err := rl.SendOutput(data); err == nil {
			return
		}
		// Relay send failed — try gRPC fallback (re-acquire lock to read onFlush)
		r.mu.RLock()
		fn := r.onFlush
		r.mu.RUnlock()
		if fn != nil {
			fn(data)
		}
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

// SetRelayClient sets the relay client reference.
// If there is buffered early output, it is replayed through the client immediately.
// Pass nil to disable relay output and fall back to gRPC.
// Thread-safe.
func (r *OutputRouter) SetRelayClient(client RelayWriter) {
	r.mu.Lock()
	r.relay = client
	buffered := r.drainEarlyBufferLocked()
	r.mu.Unlock()

	// Replay buffered data outside the lock to avoid deadlocks
	if client != nil && client.IsConnected() && len(buffered) > 0 {
		logger.Terminal().Info("OutputRouter: replaying early buffer via relay", "bytes", len(buffered))
		if err := client.SendOutput(buffered); err != nil {
			logger.Terminal().Warn("OutputRouter: failed to replay early buffer via relay, trying gRPC", "error", err)
			r.mu.RLock()
			fn := r.onFlush
			r.mu.RUnlock()
			if fn != nil {
				fn(buffered)
			}
		}
	} else if client == nil && len(buffered) > 0 {
		// Clearing relay with buffered data — send via gRPC
		r.mu.RLock()
		fn := r.onFlush
		r.mu.RUnlock()
		if fn != nil {
			logger.Terminal().Info("OutputRouter: replaying early buffer via gRPC (relay cleared)", "bytes", len(buffered))
			fn(buffered)
		}
	}
}

// HasRelayClient returns whether a relay client is configured.
func (r *OutputRouter) HasRelayClient() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relay != nil
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
