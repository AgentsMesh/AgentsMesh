package channel

import (
	"fmt"
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

// writeToPublisher writes data to the publisher connection with serialization.
// Returns nil silently if no publisher is connected (normal during reconnection).
func (c *TerminalChannel) writeToPublisher(data []byte) error {
	c.publisherWriteMu.Lock()
	defer c.publisherWriteMu.Unlock()

	c.publisherMu.RLock()
	pub := c.publisher
	c.publisherMu.RUnlock()

	if pub == nil {
		c.logger.Debug("Dropping data, publisher not connected")
		return nil
	}
	return writeToConn(pub, data)
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
	publisher        *websocket.Conn
	publisherMu      sync.RWMutex
	publisherWriteMu sync.Mutex // Serializes writes to publisher (gorilla/websocket is not concurrent-write safe)

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
// If an old publisher connection exists, it is closed so its forwarding goroutine exits cleanly.
func (c *TerminalChannel) SetPublisher(conn *websocket.Conn) {
	c.publisherMu.Lock()

	// Check closed INSIDE publisherMu to prevent race with Close().
	// Without this, Close() could complete between the closed check and
	// publisherMu.Lock(), leaving a goroutine on a dead channel.
	c.closedMu.RLock()
	if c.closed {
		c.closedMu.RUnlock()
		c.publisherMu.Unlock()
		conn.Close()
		return
	}
	c.closedMu.RUnlock()

	oldConn := c.publisher
	wasDisconnected := c.publisherDisconnected
	c.publisher = conn
	c.publisherDisconnected = false

	// Cancel reconnect timer if exists
	if c.publisherReconnectTimer != nil {
		c.publisherReconnectTimer.Stop()
		c.publisherReconnectTimer = nil
	}
	c.publisherMu.Unlock()

	// Close old publisher connection so its forwarding goroutine exits via ReadMessage error.
	// The old goroutine's handlePublisherDisconnect will see c.publisher != oldConn and return early.
	if oldConn != nil && oldConn != conn {
		oldConn.Close()
	}

	if wasDisconnected {
		c.logger.Info("Publisher reconnected")
		// Notify all subscribers that publisher has reconnected
		c.Broadcast(protocol.EncodeRunnerReconnected())
	} else {
		c.logger.Info("Publisher connected")
	}

	// Start forwarding from publisher to subscribers
	go c.forwardPublisherToSubscribers()
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

// AddSubscriber adds a subscriber (browser observer)
func (c *TerminalChannel) AddSubscriber(subscriberID string, conn *websocket.Conn) {
	_ = c.addSubscriberInternal(subscriberID, conn, 0)
}

// AddSubscriberWithLimit atomically checks subscriber count and adds if under limit.
// Returns MaxSubscribersError if at capacity. Use maxSubscribers=0 for no limit.
func (c *TerminalChannel) AddSubscriberWithLimit(subscriberID string, conn *websocket.Conn, maxSubscribers int) error {
	return c.addSubscriberInternal(subscriberID, conn, maxSubscribers)
}

// addSubscriberInternal is the shared implementation for AddSubscriber/AddSubscriberWithLimit.
func (c *TerminalChannel) addSubscriberInternal(subscriberID string, conn *websocket.Conn, maxSubscribers int) error {
	subscriber := &Subscriber{ID: subscriberID, Conn: conn}

	c.subscribersMu.Lock()

	// Check closed INSIDE subscribersMu to prevent TOCTOU race with Close().
	// Without this, Close() could complete between the closed check and
	// subscribersMu.Lock(), leaving a goroutine on a dead channel.
	c.closedMu.RLock()
	if c.closed {
		c.closedMu.RUnlock()
		c.subscribersMu.Unlock()
		conn.Close()
		return fmt.Errorf("channel closed")
	}
	c.closedMu.RUnlock()

	if maxSubscribers > 0 && len(c.subscribers) >= maxSubscribers {
		c.subscribersMu.Unlock()
		return &MaxSubscribersError{Max: maxSubscribers}
	}
	c.subscribers[subscriberID] = subscriber

	// Cancel keep-alive timer if exists
	if c.keepAliveTimer != nil {
		c.keepAliveTimer.Stop()
		c.keepAliveTimer = nil
	}
	count := len(c.subscribers)
	c.subscribersMu.Unlock()

	c.logger.Info("Subscriber connected", "subscriber_id", subscriberID, "total_subscribers", count)

	// Send buffered Output messages to new subscriber
	// This allows new observers to see recent terminal output they missed
	bufferedOutput := c.getBufferedOutput()
	if len(bufferedOutput) > 0 {
		c.logger.Debug("Sending buffered output to new subscriber",
			"subscriber_id", subscriberID, "count", len(bufferedOutput))
		for _, data := range bufferedOutput {
			if err := subscriber.WriteMessage(data); err != nil {
				c.logger.Warn("Failed to send buffered output to new subscriber",
					"subscriber_id", subscriberID, "error", err)
				break // Stop sending if connection has issues
			}
		}
	}

	// Notify new subscriber if publisher is currently disconnected
	if c.IsPublisherDisconnected() {
		if err := subscriber.WriteMessage(protocol.EncodeRunnerDisconnected()); err != nil {
			c.logger.Warn("Failed to send publisher disconnected status to new subscriber",
				"subscriber_id", subscriberID, "error", err)
		}
	}

	// Start forwarding from this subscriber to publisher
	go c.forwardSubscriberToPublisher(subscriberID)
	return nil
}

// RemoveSubscriber removes a subscriber
func (c *TerminalChannel) RemoveSubscriber(subscriberID string) {
	c.subscribersMu.Lock()
	subscriber, ok := c.subscribers[subscriberID]
	if !ok {
		c.subscribersMu.Unlock()
		return
	}
	subscriber.Conn.Close()
	delete(c.subscribers, subscriberID)
	count := len(c.subscribers)
	c.subscribersMu.Unlock()

	c.logger.Info("Subscriber disconnected", "subscriber_id", subscriberID, "remaining_subscribers", count)

	// Release control if this subscriber had it
	c.controllerMu.Lock()
	if c.controllerID == subscriberID {
		c.controllerID = ""
	}
	c.controllerMu.Unlock()

	if count == 0 {
		// Last subscriber left, start keep-alive timer
		c.subscribersMu.Lock()

		// Double-check: a new subscriber may have connected between the
		// unlock above and this re-lock, preventing a spurious timer.
		if len(c.subscribers) != 0 {
			c.subscribersMu.Unlock()
			return
		}

		// Stop any existing timer to prevent duplicates
		if c.keepAliveTimer != nil {
			c.keepAliveTimer.Stop()
		}

		c.lastSubscriberDisconnect = time.Now()
		c.keepAliveTimer = time.AfterFunc(c.config.KeepAliveDuration, func() {
			// Guard against race with Close(): if channel is already closed,
			// skip the callback to avoid spurious onAllSubscribersGone calls.
			if c.IsClosed() {
				return
			}

			// Check if still no subscribers after timeout
			c.subscribersMu.RLock()
			stillEmpty := len(c.subscribers) == 0
			c.subscribersMu.RUnlock()

			if stillEmpty {
				c.logger.Info("Keep-alive timeout, notifying backend")
				if c.onAllSubscribersGone != nil {
					c.onAllSubscribersGone(c.PodKey)
				}
			}
		})
		c.subscribersMu.Unlock()
	}
}

// SubscriberCount returns the number of connected subscribers
func (c *TerminalChannel) SubscriberCount() int {
	c.subscribersMu.RLock()
	defer c.subscribersMu.RUnlock()
	return len(c.subscribers)
}

// Broadcast sends data to all connected subscribers.
// Uses snapshot-then-write pattern: subscribers are copied under lock, then
// writes happen without holding the lock. This prevents a slow/dead subscriber
// from blocking SubscriberCount(), Stats(), and the heartbeat goroutine.
func (c *TerminalChannel) Broadcast(data []byte) {
	// Snapshot subscribers under lock
	c.subscribersMu.RLock()
	subscribers := make([]*Subscriber, 0, len(c.subscribers))
	for _, sub := range c.subscribers {
		subscribers = append(subscribers, sub)
	}
	c.subscribersMu.RUnlock()

	c.logger.Debug("Broadcasting to subscribers", "data_len", len(data), "subscriber_count", len(subscribers))

	// Write to each subscriber with deadline (no lock held)
	var failedIDs []string
	for _, subscriber := range subscribers {
		if err := subscriber.WriteMessage(data); err != nil {
			c.logger.Warn("Failed to send to subscriber, removing",
				"subscriber_id", subscriber.ID, "error", err)
			failedIDs = append(failedIDs, subscriber.ID)
		} else {
			c.logger.Debug("Sent to subscriber", "subscriber_id", subscriber.ID, "data_len", len(data))
		}
	}

	// Remove failed subscribers (acquires write lock separately)
	for _, id := range failedIDs {
		c.RemoveSubscriber(id)
	}
}

// bufferOutput adds an Output message to the ring buffer for new observers
func (c *TerminalChannel) bufferOutput(data []byte) {
	c.outputBufferMu.Lock()
	defer c.outputBufferMu.Unlock()

	dataLen := len(data)

	// Evict old messages if buffer is full (by count or size)
	for len(c.outputBuffer) >= c.config.OutputBufferCount || (c.outputBufferBytes+dataLen > c.config.OutputBufferSize && len(c.outputBuffer) > 0) {
		// Remove oldest message and nil out the slot to allow GC to reclaim it
		oldMsg := c.outputBuffer[0]
		c.outputBuffer[0] = nil // prevent memory leak from underlying array reference
		c.outputBuffer = c.outputBuffer[1:]
		c.outputBufferBytes -= len(oldMsg)
	}

	// Compact: when the underlying array is much larger than the live slice,
	// copy to a fresh slice to release the wasted capacity.
	if cap(c.outputBuffer) > len(c.outputBuffer)*3 && cap(c.outputBuffer) > c.config.OutputBufferCount {
		compacted := make([][]byte, len(c.outputBuffer), c.config.OutputBufferCount)
		copy(compacted, c.outputBuffer)
		c.outputBuffer = compacted
	}

	// Only buffer if this single message fits
	if dataLen <= c.config.OutputBufferSize {
		// Make a copy to avoid data races
		dataCopy := make([]byte, dataLen)
		copy(dataCopy, data)
		c.outputBuffer = append(c.outputBuffer, dataCopy)
		c.outputBufferBytes += dataLen
	}
}

// clearOutputBuffer removes all buffered output messages.
// Called when a Snapshot arrives — snapshot supersedes all prior output.
func (c *TerminalChannel) clearOutputBuffer() {
	c.outputBufferMu.Lock()
	defer c.outputBufferMu.Unlock()
	// Nil out references to allow GC before reslicing
	for i := range c.outputBuffer {
		c.outputBuffer[i] = nil
	}
	c.outputBuffer = c.outputBuffer[:0]
	c.outputBufferBytes = 0
}

// getBufferedOutput returns a copy of all buffered Output messages
func (c *TerminalChannel) getBufferedOutput() [][]byte {
	c.outputBufferMu.RLock()
	defer c.outputBufferMu.RUnlock()

	result := make([][]byte, len(c.outputBuffer))
	for i, data := range c.outputBuffer {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		result[i] = dataCopy
	}
	return result
}

// forwardPublisherToSubscribers forwards data from publisher to all subscribers.
// Each goroutine captures its own conn reference at start. When SetPublisher replaces
// the publisher, it closes the old conn, causing this goroutine's ReadMessage to fail
// and exit cleanly via handlePublisherDisconnect's identity check.
func (c *TerminalChannel) forwardPublisherToSubscribers() {
	c.logger.Debug("Starting forwardPublisherToSubscribers loop")

	// Capture conn once — this goroutine is bound to this specific connection
	c.publisherMu.RLock()
	conn := c.publisher
	c.publisherMu.RUnlock()

	if conn == nil {
		c.logger.Debug("Publisher conn is nil, exiting forward loop")
		return
	}

	conn.SetReadLimit(maxMessageSize)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.logger.Info("Publisher disconnected", "error", err)
			c.handlePublisherDisconnect(conn)
			break
		}

		c.logger.Debug("Received data from publisher", "data_len", len(data))

		// Buffer Output messages for new observers
		msg, _ := protocol.DecodeMessage(data)
		if msg != nil && msg.Type == protocol.MsgTypeOutput {
			c.bufferOutput(data)
		}

		// Snapshot supersedes all buffered output — clear buffer and replace with
		// the snapshot itself, so new subscribers joining after this point receive
		// the complete terminal state instead of an empty buffer.
		if msg != nil && msg.Type == protocol.MsgTypeSnapshot {
			c.clearOutputBuffer()
			c.bufferOutput(data)
		}

		c.Broadcast(data)
	}
}

// forwardSubscriberToPublisher forwards input from a subscriber to publisher
func (c *TerminalChannel) forwardSubscriberToPublisher(subscriberID string) {
	c.subscribersMu.RLock()
	subscriber, ok := c.subscribers[subscriberID]
	c.subscribersMu.RUnlock()

	if !ok {
		return
	}

	subscriber.Conn.SetReadLimit(maxMessageSize)

	for {
		_, data, err := subscriber.Conn.ReadMessage()
		if err != nil {
			c.RemoveSubscriber(subscriberID)
			break
		}

		// Parse message type
		msg, err := protocol.DecodeMessage(data)
		if err != nil {
			continue
		}

		// Handle control requests
		if msg.Type == protocol.MsgTypeControl {
			c.handleControlRequest(subscriberID, msg.Payload)
			continue
		}

		// For input, image paste, and resize messages, check control permission
		if msg.Type == protocol.MsgTypeInput || msg.Type == protocol.MsgTypeImagePaste || msg.Type == protocol.MsgTypeResize {
			if !c.CanInput(subscriberID) {
				c.logger.Debug("Input rejected, no control", "subscriber_id", subscriberID)
				continue
			}
		}

		// Handle ping/pong locally
		if msg.Type == protocol.MsgTypePing {
			if err := subscriber.WriteMessage(protocol.EncodePong()); err != nil {
				c.logger.Warn("Failed to send pong", "subscriber_id", subscriberID)
			}
			continue
		}

		// Forward to publisher (serialized via publisherWriteMu)
		c.logger.Debug("Forwarding subscriber data to publisher",
			"subscriber_id", subscriberID, "msg_type", msg.Type, "data_len", len(data))
		if err := c.writeToPublisher(data); err != nil {
			c.logger.Warn("Failed to forward to publisher", "error", err)
		}
	}
}

// handlePublisherDisconnect handles publisher disconnect with connection identity check.
// Takes disconnectedConn to verify it's still the current publisher before acting.
// If SetPublisher already replaced the publisher, this is a no-op (early return).
func (c *TerminalChannel) handlePublisherDisconnect(disconnectedConn *websocket.Conn) {
	c.publisherMu.Lock()

	// Identity check: if publisher was already replaced by SetPublisher, just return.
	// The old conn was already closed by SetPublisher, no need to close again.
	if c.publisher != disconnectedConn {
		c.publisherMu.Unlock()
		return
	}

	// Save reference, nil out under lock, then close outside lock
	// to avoid blocking other goroutines while Close() performs I/O.
	conn := c.publisher
	c.publisher = nil
	c.publisherDisconnected = true

	c.logger.Info("Publisher disconnected, waiting for reconnection",
		"timeout", c.config.PublisherReconnectTimeout)

	// Start reconnect timer
	c.publisherReconnectTimer = time.AfterFunc(c.config.PublisherReconnectTimeout, func() {
		c.publisherMu.Lock()
		stillDisconnected := c.publisherDisconnected
		c.publisherMu.Unlock()

		if stillDisconnected {
			c.logger.Info("Publisher reconnect timeout, closing channel")
			c.Close()
		}
	})
	c.publisherMu.Unlock()

	// Close connection after releasing lock
	conn.Close()

	// Broadcast AFTER releasing lock — eliminates the Unlock→Lock window
	c.Broadcast(protocol.EncodeRunnerDisconnected())
}

// handleControlRequest handles input control requests
func (c *TerminalChannel) handleControlRequest(subscriberID string, payload []byte) {
	req, err := protocol.DecodeControlRequest(payload)
	if err != nil {
		return
	}

	var response *protocol.ControlRequest

	switch req.Action {
	case "request":
		if c.RequestControl(subscriberID) {
			response = &protocol.ControlRequest{Action: "granted", Controller: subscriberID}
		} else {
			c.controllerMu.RLock()
			response = &protocol.ControlRequest{Action: "denied", Controller: c.controllerID}
			c.controllerMu.RUnlock()
		}

	case "release":
		c.ReleaseControl(subscriberID)
		response = &protocol.ControlRequest{Action: "released", Controller: ""}

	case "query":
		c.controllerMu.RLock()
		response = &protocol.ControlRequest{Action: "status", Controller: c.controllerID}
		c.controllerMu.RUnlock()

	default:
		response = &protocol.ControlRequest{Action: "error", Controller: ""}
	}

	data, _ := protocol.EncodeControlRequest(response)
	// Get connection under lock, release before writing to avoid holding lock during I/O
	c.subscribersMu.RLock()
	subscriber, ok := c.subscribers[subscriberID]
	c.subscribersMu.RUnlock()

	if ok {
		if err := subscriber.WriteMessage(data); err != nil {
			c.logger.Warn("Failed to send control response", "subscriber_id", subscriberID, "error", err)
		}
	}
}

// CanInput checks if a subscriber can send input
func (c *TerminalChannel) CanInput(subscriberID string) bool {
	c.controllerMu.RLock()
	defer c.controllerMu.RUnlock()

	// No controller or this subscriber is controller
	return c.controllerID == "" || c.controllerID == subscriberID
}

// RequestControl requests input control
func (c *TerminalChannel) RequestControl(subscriberID string) bool {
	c.controllerMu.Lock()
	defer c.controllerMu.Unlock()

	if c.controllerID == "" {
		c.controllerID = subscriberID
		c.logger.Info("Control granted", "subscriber_id", subscriberID)
		return true
	}
	return false
}

// ReleaseControl releases input control
func (c *TerminalChannel) ReleaseControl(subscriberID string) {
	c.controllerMu.Lock()
	defer c.controllerMu.Unlock()

	if c.controllerID == subscriberID {
		c.controllerID = ""
		c.logger.Info("Control released", "subscriber_id", subscriberID)
	}
}

// Close closes the channel and all connections
func (c *TerminalChannel) Close() {
	c.closedMu.Lock()
	if c.closed {
		c.closedMu.Unlock()
		return
	}
	c.closed = true
	c.closedMu.Unlock()

	c.logger.Info("Closing channel")

	// Stop keep-alive timer
	c.subscribersMu.Lock()
	if c.keepAliveTimer != nil {
		c.keepAliveTimer.Stop()
	}
	c.subscribersMu.Unlock()

	// Stop publisher reconnect timer and close publisher connection
	c.publisherMu.Lock()
	if c.publisherReconnectTimer != nil {
		c.publisherReconnectTimer.Stop()
		c.publisherReconnectTimer = nil
	}
	if c.publisher != nil {
		c.publisher.Close()
		c.publisher = nil
	}
	c.publisherMu.Unlock()

	// Close all subscriber connections
	c.subscribersMu.Lock()
	for _, subscriber := range c.subscribers {
		subscriber.Conn.Close()
	}
	c.subscribers = make(map[string]*Subscriber)
	c.subscribersMu.Unlock()

	// Release output buffer memory
	c.outputBufferMu.Lock()
	for i := range c.outputBuffer {
		c.outputBuffer[i] = nil
	}
	c.outputBuffer = nil
	c.outputBufferBytes = 0
	c.outputBufferMu.Unlock()

	// Notify channel closed
	if c.onChannelClosed != nil {
		c.onChannelClosed(c.PodKey)
	}
}

// IsClosed checks if the channel is closed
func (c *TerminalChannel) IsClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}
