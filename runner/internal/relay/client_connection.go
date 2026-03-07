package relay

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/gorilla/websocket"
)

// Connect establishes connection to the Relay server.
func (c *Client) Connect() error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	if err := c.connectInternal(); err != nil {
		return err
	}

	// Mark as connected after successful dial
	// For reconnectLoop, connected is set inside wgMu lock to prevent race with Stop()
	c.connected.Store(true)
	c.connectedAt.Store(time.Now().UnixMilli())
	return nil
}

func (c *Client) connectInternal() error {
	// Snapshot token under connMu to avoid data race with UpdateToken()
	c.connMu.RLock()
	token := c.token
	c.connMu.RUnlock()

	// Build WebSocket URL with query parameters
	u, err := url.Parse(c.relayURL)
	if err != nil {
		return fmt.Errorf("invalid relay URL: %w", err)
	}

	// Convert HTTP/HTTPS to WS/WSS
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// Already correct
	default:
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Append endpoint path to the base URL path (e.g., /relay -> /relay/runner/terminal)
	// This preserves any path prefix from reverse proxy configuration
	u.Path = path.Join(u.Path, "/runner/terminal")
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()

	// Log URL without token for security
	c.logger.Info("Connecting to relay", "host", u.Host, "path", u.Path)

	// Establish WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}

	conn, _, err := dialer.DialContext(c.ctx, u.String(), nil)
	if err != nil {
		c.logger.Error("WebSocket dial failed", "host", u.Host, "error", err)
		return fmt.Errorf("websocket dial failed: %w", err)
	}

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// Note: connected and connectedAt are set by the caller (Connect or reconnectLoop)
	// to ensure proper synchronization with Stop(). Do NOT set them here.

	c.logger.Info("Connected to relay successfully")
	return nil
}

// Start starts the read and write loops
// Returns false if client is already stopped
func (c *Client) Start() bool {
	c.wgMu.Lock()
	defer c.wgMu.Unlock()

	// Check if already stopped before adding to WaitGroup
	// This check + wg.Add must be atomic to prevent race with Stop()
	if c.stopped.Load() {
		c.logger.Debug("Client already stopped, not starting loops")
		return false
	}

	c.logger.Info("Starting relay client loops")
	c.wg.Add(2)
	safego.Go("relay-read", c.readLoop)
	safego.Go("relay-write", c.writeLoop)
	return true
}

// Stop gracefully closes the connection
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		c.logger.Info("Stopping relay client")

		// Acquire wgMu to ensure no new wg.Add() can happen while we set stopped=true
		// This guarantees that after this block, no new goroutines will be added to wg
		c.wgMu.Lock()
		c.stopped.Store(true)
		c.wgMu.Unlock()

		// Mark as disconnected so HasRelayClient() returns false
		c.connected.Store(false)

		close(c.stopCh)
		c.cancel()

		// Close connection to unblock readLoop immediately
		c.connMu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.connMu.Unlock()

		// Wait for read/write loops to exit with timeout
		// Since stopped=true and wgMu was held, no new loops can be started
		done := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Normal exit
		case <-time.After(5 * time.Second):
			c.logger.Warn("Timeout waiting for relay loops to exit")
		}

		// Clean up connection reference
		c.connMu.Lock()
		c.conn = nil
		c.connMu.Unlock()

		c.logger.Info("Relay client stopped")
	})
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}
