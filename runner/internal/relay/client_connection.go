package relay

import (
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

// Connect establishes connection to the Relay server
func (c *Client) Connect() error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	return c.connectInternal()
}

func (c *Client) connectInternal() error {
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
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	// Log URL without token for security
	c.logger.Info("Connecting to relay", "host", u.Host, "path", u.Path)

	// Establish WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
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
	c.connected.Store(true)
	c.connectedAt.Store(time.Now().UnixMilli())

	c.logger.Info("Connected to relay successfully")
	return nil
}

// Start starts the read and write loops
func (c *Client) Start() {
	c.logger.Info("Starting relay client loops")
	c.wg.Add(2)
	go c.readLoop()
	go c.writeLoop()
}

// Stop gracefully closes the connection
func (c *Client) Stop() {
	c.stopOnce.Do(func() {
		c.logger.Info("Stopping relay client")

		// Mark as disconnected immediately so HasRelayClient() returns false
		// This allows new subscribe_terminal to create a fresh connection
		c.connected.Store(false)

		close(c.stopCh)
		c.cancel()

		// Close connection to unblock readLoop immediately
		c.connMu.Lock()
		if c.conn != nil {
			// Close connection to interrupt any pending reads
			c.conn.Close()
		}
		c.connMu.Unlock()

		// Wait for read/write loops to exit with timeout
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
