package channel

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// writeWait is the maximum time allowed for a WebSocket write to complete.
// Prevents dead/slow connections from blocking indefinitely and causing lock contention.
const writeWait = 5 * time.Second

// maxMessageSize limits the maximum incoming WebSocket message size (4MB).
// Prevents a malicious or buggy client from causing OOM via oversized frames.
const maxMessageSize = 4 * 1024 * 1024

// writeToConn writes a binary message to a WebSocket connection with a write deadline.
func writeToConn(conn *websocket.Conn, data []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// Subscriber represents a browser WebSocket connection (observer)
type Subscriber struct {
	ID      string
	Conn    *websocket.Conn
	writeMu sync.Mutex // Serializes writes (gorilla/websocket is not concurrent-write safe)
}

// WriteMessage writes a binary message to the subscriber with a write deadline.
// Serializes concurrent writes since gorilla/websocket does not support them.
func (s *Subscriber) WriteMessage(data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return writeToConn(s.Conn, data)
}

// ChannelConfig holds configuration for a terminal channel
type ChannelConfig struct {
	KeepAliveDuration         time.Duration // How long to keep channel alive after all subscribers disconnect
	PublisherReconnectTimeout time.Duration // How long to wait for publisher (runner) to reconnect
	SubscriberReconnectTimeout time.Duration // How long to wait for subscriber to reconnect (future use)
	OutputBufferSize          int           // Max bytes for output buffer
	OutputBufferCount         int           // Max messages for output buffer
}

// DefaultChannelConfig returns default channel configuration
func DefaultChannelConfig() ChannelConfig {
	return ChannelConfig{
		KeepAliveDuration:         30 * time.Second,
		PublisherReconnectTimeout: 30 * time.Second,
		SubscriberReconnectTimeout: 30 * time.Second,
		OutputBufferSize:          256 * 1024, // 256KB
		OutputBufferCount:         200,
	}
}

// TerminalChannel manages a terminal channel between Runner (publisher) and multiple Browsers (subscribers)
// This follows the producer-consumer / observer pattern:
// - One Runner as Publisher (producer)
// - Multiple Browsers as Subscribers (observers)
// - Channel identified by PodKey (not session ID)
type TerminalChannel struct {
	PodKey string // Channel unique identifier

	// Configuration
	config ChannelConfig

	// Publisher: Runner connection (1)
	publisher          *websocket.Conn
	publisherMu        sync.RWMutex
	publisherWriteMu   sync.Mutex // Serializes writes to publisher (gorilla/websocket is not concurrent-write safe)
	publisherReplaceMu sync.Mutex // Serializes entire SetPublisher replacement sequence

	// Subscribers: Browser connections (N)
	subscribers   map[string]*Subscriber // subscriberID -> conn
	subscribersMu sync.RWMutex

	// Disconnect handling
	lastSubscriberDisconnect time.Time
	keepAliveTimer           *time.Timer

	// Input control
	controllerID string // Current controller subscriber ID
	controllerMu sync.RWMutex

	// Output buffer for new observers (ring buffer of recent Output messages)
	// This allows new subscribers to receive recent terminal output missed before connecting
	outputBuffer      [][]byte
	outputBufferBytes int // Total bytes in buffer (for size limiting)
	outputBufferMu    sync.RWMutex

	// Publisher reconnection support
	publisherDisconnected   bool        // Publisher currently disconnected
	publisherReconnectTimer *time.Timer // Timer for publisher reconnect timeout

	// Publisher goroutine lifecycle
	publisherEpoch uint64         // Incremented each SetPublisher call
	publisherWg    sync.WaitGroup // Tracks active forwardPublisherToSubscribers goroutine

	// Callbacks
	onAllSubscribersGone func(podKey string)
	onChannelClosed      func(podKey string)

	// Channel state
	closed   bool
	closedMu sync.RWMutex

	logger *slog.Logger
}

// NewTerminalChannel creates a new terminal channel with default configuration
func NewTerminalChannel(podKey string, keepAliveDuration time.Duration, onAllSubscribersGone func(string), onChannelClosed func(string)) *TerminalChannel {
	cfg := DefaultChannelConfig()
	cfg.KeepAliveDuration = keepAliveDuration
	return NewTerminalChannelWithConfig(podKey, cfg, onAllSubscribersGone, onChannelClosed)
}

// NewTerminalChannelWithConfig creates a new terminal channel with custom configuration
func NewTerminalChannelWithConfig(podKey string, cfg ChannelConfig, onAllSubscribersGone func(string), onChannelClosed func(string)) *TerminalChannel {
	return &TerminalChannel{
		PodKey:               podKey,
		config:               cfg,
		subscribers:          make(map[string]*Subscriber),
		onAllSubscribersGone: onAllSubscribersGone,
		onChannelClosed:      onChannelClosed,
		outputBuffer:         make([][]byte, 0, cfg.OutputBufferCount),
		logger:               slog.With("pod_key", podKey),
	}
}

// SetPublisher sets the publisher (runner) connection.
// If an old publisher connection exists, it is closed and its forwarding goroutine
// is awaited before starting a new one, eliminating concurrent goroutine races.
// Serialized via publisherReplaceMu to prevent WaitGroup races on concurrent calls.
func (c *TerminalChannel) SetPublisher(conn *websocket.Conn) {
	c.publisherReplaceMu.Lock()
	defer c.publisherReplaceMu.Unlock()

	c.publisherMu.Lock()

	// Check closed INSIDE publisherMu to prevent race with Close().
	// Without this, Close() could complete between the closed check and
	// publisherMu.Lock(), leaving a goroutine on a dead channel.
	c.closedMu.RLock()
	if c.closed {
		c.closedMu.RUnlock()
		c.publisherMu.Unlock()
		_ = conn.Close()
		return
	}
	c.closedMu.RUnlock()

	oldConn := c.publisher
	wasDisconnected := c.publisherDisconnected

	// Same-conn guard: if the caller passes the exact same connection pointer,
	// skip the replacement entirely — closing oldConn would kill the active
	// goroutine, and publisherWg.Wait() would block forever.
	if oldConn == conn {
		c.publisherMu.Unlock()
		return
	}

	c.publisher = conn
	c.publisherDisconnected = false
	c.publisherEpoch++
	epoch := c.publisherEpoch

	// Cancel reconnect timer if exists
	if c.publisherReconnectTimer != nil {
		c.publisherReconnectTimer.Stop()
		c.publisherReconnectTimer = nil
	}
	c.publisherMu.Unlock()

	// Close old publisher connection so its forwarding goroutine exits via ReadMessage error.
	if oldConn != nil {
		_ = oldConn.Close()
	}

	// Wait for old goroutine to exit before starting new one — eliminates
	// the race window where two forwarding goroutines run concurrently.
	c.publisherWg.Wait()

	if wasDisconnected {
		c.logger.Info("Publisher reconnected")
		// Notify all subscribers that publisher has reconnected
		c.Broadcast(protocol.EncodeRunnerReconnected())
	} else {
		c.logger.Info("Publisher connected")
	}

	// Start forwarding from publisher to subscribers
	c.publisherWg.Add(1)
	go c.forwardPublisherToSubscribers(epoch)
}

// GetPublisher returns the publisher connection (for checking if connected)
func (c *TerminalChannel) GetPublisher() *websocket.Conn {
	c.publisherMu.RLock()
	defer c.publisherMu.RUnlock()
	return c.publisher
}

// IsPublisherDisconnected returns true if the publisher is currently disconnected
func (c *TerminalChannel) IsPublisherDisconnected() bool {
	c.publisherMu.RLock()
	defer c.publisherMu.RUnlock()
	return c.publisherDisconnected
}

// IsClosed checks if the channel is closed
func (c *TerminalChannel) IsClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}
