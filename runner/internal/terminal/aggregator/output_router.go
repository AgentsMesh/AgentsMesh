// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"sync"
)

// RelayWriter abstracts the relay client for output routing.
// Route() checks IsConnected() at call time, eliminating stale-closure races.
type RelayWriter interface {
	SendOutput(data []byte) error
	IsConnected() bool
}

// OutputRouter routes terminal output to the Relay WebSocket destination.
//
// When Relay is connected, output is sent via WebSocket.
// When Relay is not connected (no subscriber, or during reconnect), output is
// silently dropped — this is expected behavior since no one is observing the terminal.
type OutputRouter struct {
	mu    sync.RWMutex
	relay RelayWriter // Relay client reference (checked at Route time)
}

// NewOutputRouter creates a new output router.
func NewOutputRouter() *OutputRouter {
	return &OutputRouter{}
}

// Route sends data to the Relay if connected, otherwise drops it.
//
// This checks relay.IsConnected() at call time so stale closures from old
// relay clients cannot intercept output. When relay is disconnected (e.g.,
// during reconnect or no subscriber), output is dropped — this is fine
// because no one is observing the terminal.
//
// This method is safe to call from any goroutine.
func (r *OutputRouter) Route(data []byte) {
	if len(data) == 0 {
		return
	}

	r.mu.RLock()
	relay := r.relay
	r.mu.RUnlock()

	if relay != nil && relay.IsConnected() {
		_ = relay.SendOutput(data)
	}
	// No relay or not connected: output is dropped.
	// This is expected — terminal output is only meaningful when someone is watching.
}

// SetRelayClient sets the relay client reference.
// Pass nil to clear the relay client.
// Thread-safe.
func (r *OutputRouter) SetRelayClient(client RelayWriter) {
	r.mu.Lock()
	r.relay = client
	r.mu.Unlock()
}

// HasRelayClient returns whether a relay client is configured.
func (r *OutputRouter) HasRelayClient() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relay != nil
}
