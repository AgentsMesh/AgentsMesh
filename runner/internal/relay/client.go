package relay

import (
	"context"
	"log/slog"
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
	maxMessageSize = 4 * 1024 * 1024 // 4MB max message size

	// Reconnection settings
	maxReconnectDelay  = 30 * time.Second
	initialBackoff     = 500 * time.Millisecond
	minStableConnected = 10 * time.Second // connection must last this long to reset backoff
)

// CloseHandler is called when the connection is closed
type CloseHandler func()

// Client is a WebSocket client for connecting to the Relay service
// Note: sessionID has been removed - channels are now identified by PodKey only
type Client struct {
	// Configuration
	relayURL string
	podKey   string
	token    string // JWT token for authentication

	// WebSocket connection
	conn   *websocket.Conn
	connMu sync.RWMutex

	// Handlers
	handlers       map[byte]func([]byte)
	handlersMu     sync.RWMutex
	onClose        CloseHandler
	onReconnect    func()                   // Called after successful reconnection
	onTokenExpired func() (newToken string) // Called when token expires, should request new token from Backend

	// State
	connected      atomic.Bool
	connectedAt    atomic.Int64  // Unix milliseconds timestamp when connected
	reconnecting   atomic.Bool   // Prevents concurrent reconnect attempts
	reconnectCount atomic.Int32  // Tracks consecutive short-lived connections (flap detection)
	stopped        atomic.Bool   // Indicates client has been permanently stopped
	stopCh         chan struct{} // Signals client shutdown (permanent)
	connDoneCh     chan struct{} // Signals current connection is done (closed on disconnect)
	stopOnce       sync.Once
	closeOnce      sync.Once // Ensures onClose callback fires at most once
	sendCh         chan []byte
	logger         *slog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	wgMu           sync.Mutex // Protects wg.Add() to ensure atomicity with stopped check
	reconnectMu    sync.Mutex
}

// NewClient creates a new Relay WebSocket client
// Note: sessionID parameter has been removed - channels are identified by PodKey only
func NewClient(relayURL, podKey, token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		relayURL:   relayURL,
		podKey:     podKey,
		token:      token,
		stopCh:     make(chan struct{}),
		connDoneCh: make(chan struct{}),
		sendCh:     make(chan []byte, 256), // Buffered send channel
		handlers:   make(map[byte]func([]byte)),
		logger: logger.With(
			"component", "relay_client",
			"pod_key", podKey,
		),
		ctx:    ctx,
		cancel: cancel,
	}

	client.logger.Info("Relay client created", "relay_url", relayURL)
	return client
}

// SetMessageHandler registers a handler for a specific message type.
func (c *Client) SetMessageHandler(msgType byte, handler func(payload []byte)) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[msgType] = handler
}

// SetCloseHandler sets the handler for connection close events
func (c *Client) SetCloseHandler(handler CloseHandler) {
	c.onClose = handler
}

// fireOnClose invokes the onClose callback exactly once, regardless of how many
// code paths trigger it (readLoop defer, reconnectLoop failure branches, etc.).
func (c *Client) fireOnClose() {
	c.closeOnce.Do(func() {
		if c.onClose != nil {
			c.onClose()
		}
	})
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

// GetRelayURL returns the relay URL
func (c *Client) GetRelayURL() string {
	return c.relayURL
}

// GetConnectedAt returns the timestamp (Unix milliseconds) when the connection was established
func (c *Client) GetConnectedAt() int64 {
	return c.connectedAt.Load()
}
