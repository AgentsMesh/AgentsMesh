package relay

import (
	"math/rand"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// isHandshakeError checks if the error is a WebSocket handshake failure
// which typically indicates token expiration or authentication issues
func isHandshakeError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "bad handshake") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403")
}

const (
	// maxConsecutiveAuthFailures is the circuit-breaker threshold for permanent
	// authentication errors. After this many consecutive handshake failures
	// (where token refresh also fails), the reconnect loop gives up.
	// This prevents zombie connections for dead/expired pods.
	maxConsecutiveAuthFailures = 5
)

// reconnectLoop attempts to reconnect to the relay server with exponential backoff.
// It is tracked by wg so that Stop() reliably waits for it to exit.
func (c *Client) reconnectLoop() {
	defer c.wg.Done()
	defer c.reconnecting.Store(false)

	// Check if client is already stopped - no point in reconnecting
	if c.stopped.Load() {
		c.logger.Debug("Client stopped, skipping reconnect")
		c.fireOnClose()
		return
	}

	// First, ensure the old connection is properly closed
	// The connDoneCh is already closed by readLoop's defer, which signals writeLoop to exit
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	// Wait for writeLoop to exit. readLoop already exited (we're spawned from its defer).
	// connDoneCh was closed by readLoop, which tells writeLoop to return.
	// writeExitCh is closed by writeLoop on exit — we wait on it directly,
	// avoiding wg.Wait() which would deadlock (reconnectLoop is in the same wg).
	writeExit := c.writeExitCh
	select {
	case <-writeExit:
		// writeLoop has fully exited
	case <-c.stopCh:
		c.logger.Info("Reconnect cancelled while waiting for loops, client stopped")
		c.fireOnClose()
		return
	case <-time.After(2 * time.Second):
		c.logger.Warn("Timeout waiting for writeLoop to exit before reconnect, aborting")
		c.fireOnClose()
		return
	}

	// Use reconnectCount to resume backoff across reconnectLoop invocations.
	// When a connection "succeeds" but dies immediately (flap), readLoop increments
	// reconnectCount. We use it here so the backoff doesn't reset to 500ms each time.
	flapCount := int(c.reconnectCount.Load())
	backoff := initialBackoff
	for i := 0; i < flapCount; i++ {
		backoff = min(backoff*2, maxReconnectDelay)
	}
	if flapCount > 0 {
		c.logger.Info("Applying flap-aware backoff",
			"flap_count", flapCount, "initial_backoff", backoff)
	}
	tokenRefreshAttempted := false
	consecutiveAuthFailures := 0

	for attempt := 1; ; attempt++ {
		// Check if Stop() was called during reconnection
		select {
		case <-c.stopCh:
			c.logger.Info("Reconnect cancelled, client stopped")
			c.fireOnClose()
			return
		case <-c.ctx.Done():
			c.logger.Info("Reconnect cancelled, context done")
			c.fireOnClose()
			return
		case <-time.After(backoff):
			// Wait before attempting reconnection
		}

		c.logger.Info("Attempting to reconnect to relay",
			"attempt", attempt,
			"backoff", backoff)

		c.reconnectMu.Lock()
		err := c.connectInternal()
		c.reconnectMu.Unlock()

		if err != nil {
			c.logger.Warn("Reconnect failed",
				"error", err,
				"attempt", attempt,
				"next_backoff", min(backoff*2, maxReconnectDelay))

			if isHandshakeError(err) {
				consecutiveAuthFailures++

				// Try to refresh token once on first auth failure
				if !tokenRefreshAttempted && c.onTokenExpired != nil {
					tokenRefreshAttempted = true
					c.logger.Info("Handshake failed, requesting new token from Backend")

					newToken := c.onTokenExpired()
					if newToken != "" {
						c.logger.Info("Received new token, retrying connection")
						c.UpdateToken(newToken)
						consecutiveAuthFailures = 0
						continue
					}
					c.logger.Warn("Failed to get new token, continuing with exponential backoff")
				}

				// Circuit breaker: give up after too many consecutive auth failures.
				// This prevents zombie reconnect loops for dead/expired pods.
				if consecutiveAuthFailures >= maxConsecutiveAuthFailures {
					c.logger.Error("Giving up reconnect: too many consecutive auth failures",
						"count", consecutiveAuthFailures, "last_error", err)
					c.fireOnClose()
					return
				}
			} else {
				// Transient error (network, timeout) — reset auth failure counter
				consecutiveAuthFailures = 0
			}

			// Exponential backoff with jitter (±20%) to prevent thundering herd
			baseBackoff := min(backoff*2, maxReconnectDelay)
			jitter := time.Duration(float64(baseBackoff) * (rand.Float64()*0.4 - 0.2))
			backoff = baseBackoff + jitter
			continue
		}

		c.logger.Info("Reconnected to relay successfully")

		// Atomically check stopped and mark connected inside wgMu lock.
		// This prevents race condition where Stop() sets stopped=true and
		// connected=false between connectInternal() and our check here.
		// Without this lock, connectInternal succeeds → Stop() runs →
		// connected is left as true when the test checks IsConnected().
		c.wgMu.Lock()
		if c.stopped.Load() {
			c.wgMu.Unlock()
			c.logger.Info("Client stopped during reconnection, closing new connection")
			c.connMu.Lock()
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.connMu.Unlock()
			c.fireOnClose()
			return
		}

		// Mark as connected only after confirming not stopped (under wgMu lock)
		c.connected.Store(true)
		c.connectedAt.Store(time.Now().UnixMilli())

		// Create new per-connection channels for the new connection
		c.connDoneCh = make(chan struct{})
		c.writeExitCh = make(chan struct{})

		// Restart read/write loops
		c.wg.Add(2)
		c.wgMu.Unlock()

		safego.Go("relay-read", c.readLoop)
		safego.Go("relay-write", c.writeLoop)

		// Trigger reconnect callback (e.g., to resend snapshot)
		// Defense-in-depth: check stopped again before firing callback.
		// Even though the architecture (reference-based OutputRouter) now prevents
		// stale callbacks from causing damage, this guard ensures no callback
		// runs after Stop() as an additional safety net.
		if c.onReconnect != nil && !c.stopped.Load() {
			c.onReconnect()
		}
		return
	}
}
