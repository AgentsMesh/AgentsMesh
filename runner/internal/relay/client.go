package relay

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Connection timeouts
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB max message size

	// Reconnection settings
	maxReconnectDelay = 30 * time.Second
	initialBackoff    = 500 * time.Millisecond
)

// InputHandler is called when user input is received from relay
type InputHandler func(data []byte)

// ResizeHandler is called when terminal resize request is received from relay
type ResizeHandler func(cols, rows uint16)

// CloseHandler is called when the connection is closed
type CloseHandler func()

// Client is a WebSocket client for connecting to the Relay service
type Client struct {
	// Configuration
	relayURL  string
	podKey    string
	sessionID string
	token     string // JWT token for authentication

	// WebSocket connection
	conn   *websocket.Conn
	connMu sync.RWMutex

	// Handlers
	onInput        InputHandler
	onResize       ResizeHandler
	onClose        CloseHandler
	onReconnect    func()                   // Called after successful reconnection
	onTokenExpired func() (newToken string) // Called when token expires, should request new token from Backend

	// State
	connected    atomic.Bool
	reconnecting atomic.Bool   // Prevents concurrent reconnect attempts
	stopCh       chan struct{} // Signals client shutdown (permanent)
	connDoneCh   chan struct{} // Signals current connection is done (closed on disconnect)
	stopOnce     sync.Once
	sendCh       chan []byte
	logger       *slog.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	reconnectMu  sync.Mutex
}

// NewClient creates a new Relay WebSocket client
func NewClient(relayURL, podKey, sessionID, token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		relayURL:   relayURL,
		podKey:     podKey,
		sessionID:  sessionID,
		token:      token,
		stopCh:     make(chan struct{}),
		connDoneCh: make(chan struct{}),
		sendCh:     make(chan []byte, 256), // Buffered send channel
		logger: logger.With(
			"component", "relay_client",
			"pod_key", podKey,
			"session_id", sessionID,
		),
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetInputHandler sets the handler for user input from browsers
func (c *Client) SetInputHandler(handler InputHandler) {
	c.onInput = handler
}

// SetResizeHandler sets the handler for terminal resize requests
func (c *Client) SetResizeHandler(handler ResizeHandler) {
	c.onResize = handler
}

// SetCloseHandler sets the handler for connection close events
func (c *Client) SetCloseHandler(handler CloseHandler) {
	c.onClose = handler
}

// SetReconnectHandler sets the handler called after successful reconnection
func (c *Client) SetReconnectHandler(handler func()) {
	c.onReconnect = handler
}

// SetTokenExpiredHandler sets the handler called when token expires during reconnection
// The handler should request a new token from Backend and return it
// If the handler returns an empty string, reconnection will continue with the old token
func (c *Client) SetTokenExpiredHandler(handler func() string) {
	c.onTokenExpired = handler
}

// UpdateToken updates the JWT token used for authentication
// This is called after receiving a new token from Backend
func (c *Client) UpdateToken(newToken string) {
	c.connMu.Lock()
	c.token = newToken
	c.connMu.Unlock()
	c.logger.Info("Token updated")
}

// GetSessionID returns the session ID
func (c *Client) GetSessionID() string {
	return c.sessionID
}

// GetRelayURL returns the relay URL
func (c *Client) GetRelayURL() string {
	return c.relayURL
}

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

	// Set path and query
	// Use JWT token for authentication instead of raw pod_key/session_id
	u.Path = "/runner/terminal"
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

	c.logger.Info("Connected to relay successfully")
	return nil
}

// Start starts the read and write loops
func (c *Client) Start() {
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
	})
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// SendSnapshot sends a terminal snapshot to the relay
func (c *Client) SendSnapshot(snapshot *TerminalSnapshot) error {
	data, err := EncodeSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	return c.send(data)
}

// SendOutput sends terminal output to the relay
func (c *Client) SendOutput(data []byte) error {
	return c.send(EncodeOutput(data))
}

// SendPong sends a pong response
func (c *Client) SendPong() error {
	return c.send(EncodePong())
}

func (c *Client) send(data []byte) error {
	if !c.connected.Load() {
		return fmt.Errorf("not connected")
	}

	select {
	case c.sendCh <- data:
		return nil
	default:
		// Channel full, drop the message
		c.logger.Warn("Send channel full, dropping message")
		return fmt.Errorf("send buffer full")
	}
}

func (c *Client) readLoop() {
	defer c.wg.Done()
	defer func() {
		c.connected.Store(false)

		// Signal writeLoop that this connection is done
		// Safe to close multiple times via select
		select {
		case <-c.connDoneCh:
			// Already closed
		default:
			close(c.connDoneCh)
		}

		// Check if this is a graceful shutdown (Stop() called) or unexpected disconnect
		select {
		case <-c.stopCh:
			// Graceful shutdown - call onClose and don't reconnect
			if c.onClose != nil {
				c.onClose()
			}
		default:
			// Unexpected disconnect - attempt reconnection
			// Use atomic.Swap to prevent concurrent reconnect attempts
			if !c.reconnecting.Swap(true) {
				go c.reconnectLoop()
			}
		}
	}()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			return
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(pongWait))

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info("Connection closed normally")
			} else {
				c.logger.Error("Read error", "error", err)
			}
			return
		}

		if messageType != websocket.BinaryMessage && messageType != websocket.TextMessage {
			continue
		}

		c.handleMessage(data)
	}
}

func (c *Client) writeLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return

		case <-c.connDoneCh:
			// Connection is done (readLoop exited), stop writeLoop
			return

		case data := <-c.sendCh:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				c.logger.Error("Write error", "error", err)
				return
			}

		case <-ticker.C:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error("Ping error", "error", err)
				return
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	msg, err := DecodeMessage(data)
	if err != nil {
		c.logger.Error("Failed to decode message", "error", err)
		return
	}

	switch msg.Type {
	case MsgTypeInput:
		if c.onInput != nil {
			c.onInput(msg.Payload)
		}

	case MsgTypeResize:
		if c.onResize != nil {
			cols, rows, err := DecodeResize(msg.Payload)
			if err != nil {
				c.logger.Error("Failed to decode resize", "error", err)
				return
			}
			c.onResize(cols, rows)
		}

	case MsgTypePing:
		// Respond with pong
		c.SendPong()

	case MsgTypePong:
		// Received pong, connection is alive

	case MsgTypeControl:
		// Control messages are not expected from relay to runner
		c.logger.Debug("Received control message (ignored)")

	default:
		c.logger.Warn("Unknown message type", "type", msg.Type)
	}
}

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

// reconnectLoop attempts to reconnect to the relay server with exponential backoff
func (c *Client) reconnectLoop() {
	defer c.reconnecting.Store(false)

	// First, ensure the old connection is properly closed
	// The connDoneCh is already closed by readLoop's defer, which signals writeLoop to exit
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	// Wait for the writeLoop to exit (readLoop already exited since we're in reconnectLoop)
	// writeLoop should exit quickly since connDoneCh is closed
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Loops exited
	case <-time.After(2 * time.Second):
		c.logger.Warn("Timeout waiting for loops to exit before reconnect")
	case <-c.stopCh:
		c.logger.Info("Reconnect cancelled while waiting for loops, client stopped")
		if c.onClose != nil {
			c.onClose()
		}
		return
	}

	backoff := initialBackoff
	const maxAttempts = 10
	tokenRefreshAttempted := false

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check if Stop() was called during reconnection
		select {
		case <-c.stopCh:
			c.logger.Info("Reconnect cancelled, client stopped")
			if c.onClose != nil {
				c.onClose()
			}
			return
		case <-c.ctx.Done():
			c.logger.Info("Reconnect cancelled, context done")
			if c.onClose != nil {
				c.onClose()
			}
			return
		case <-time.After(backoff):
			// Wait before attempting reconnection
		}

		c.logger.Info("Attempting to reconnect to relay",
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"backoff", backoff)

		c.reconnectMu.Lock()
		err := c.connectInternal()
		c.reconnectMu.Unlock()

		if err != nil {
			c.logger.Warn("Reconnect failed",
				"error", err,
				"attempt", attempt,
				"next_backoff", min(backoff*2, maxReconnectDelay))

			// Check if this is a handshake error (likely token expired)
			// Try to refresh token once
			if isHandshakeError(err) && !tokenRefreshAttempted && c.onTokenExpired != nil {
				tokenRefreshAttempted = true
				c.logger.Info("Handshake failed, requesting new token from Backend")

				// Request new token from Backend
				// This is a blocking call that waits for Backend to respond
				newToken := c.onTokenExpired()
				if newToken != "" {
					c.logger.Info("Received new token, retrying connection")
					c.UpdateToken(newToken)
					// Don't increment backoff, retry immediately with new token
					continue
				}
				c.logger.Warn("Failed to get new token, continuing with exponential backoff")
			}

			backoff = min(backoff*2, maxReconnectDelay)
			continue
		}

		c.logger.Info("Reconnected to relay successfully")

		// Create a new connDoneCh for the new connection
		c.connDoneCh = make(chan struct{})

		// Restart read/write loops
		c.wg.Add(2)
		go c.readLoop()
		go c.writeLoop()

		// Trigger reconnect callback (e.g., to resend snapshot)
		if c.onReconnect != nil {
			c.onReconnect()
		}
		return
	}

	// Failed to reconnect after max attempts - give up and call onClose
	c.logger.Error("Failed to reconnect after max attempts",
		"max_attempts", maxAttempts)
	if c.onClose != nil {
		c.onClose()
	}
}
