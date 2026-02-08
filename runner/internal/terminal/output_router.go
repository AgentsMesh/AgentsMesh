// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// OutputRouter routes terminal output to appropriate destinations.
// Supports both legacy gRPC output and modern Relay WebSocket output.
//
// Priority: Relay > gRPC (when Relay is connected, gRPC is not used)
type OutputRouter struct {
	mu          sync.RWMutex
	onFlush     func([]byte) // gRPC fallback callback
	relayOutput func([]byte) // Relay preferred callback
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
// Priority: Relay > gRPC
//
// This method is safe to call from any goroutine.
func (r *OutputRouter) Route(data []byte) {
	if len(data) == 0 {
		return
	}

	r.mu.RLock()
	relayOutput := r.relayOutput
	onFlush := r.onFlush
	r.mu.RUnlock()

	// Priority: Relay mode > Legacy gRPC mode
	// When Relay is connected, send data ONLY through Relay (not gRPC)
	// This avoids duplicate data transmission and reduces Backend load
	log := logger.TerminalTrace()
	if relayOutput != nil {
		log.Trace("OutputRouter: sending to relay", "bytes", len(data))
		relayOutput(data)
	} else if onFlush != nil {
		// Legacy fallback: send through gRPC when no Relay connected
		// This ensures backward compatibility during migration
		log.Trace("OutputRouter: sending to gRPC (no relay)", "bytes", len(data))
		onFlush(data)
	} else {
		log.Warn("OutputRouter: no output callback set", "bytes", len(data))
	}
}

// SetRelayOutput sets the relay output callback.
// Pass nil to disable relay output and fall back to gRPC.
// Thread-safe.
func (r *OutputRouter) SetRelayOutput(fn func([]byte)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.relayOutput = fn
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
func (r *OutputRouter) SetOnFlush(fn func([]byte)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onFlush = fn
}
