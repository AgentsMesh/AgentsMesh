package channel

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// Subscriber represents a browser WebSocket connection (observer)
type Subscriber struct {
	ID   string
	Conn *websocket.Conn
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
	publisher   *websocket.Conn
	publisherMu sync.RWMutex

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

// SetPublisher sets the publisher (runner) connection
func (c *TerminalChannel) SetPublisher(conn *websocket.Conn) {
	c.publisherMu.Lock()
	wasDisconnected := c.publisherDisconnected
	c.publisher = conn
	c.publisherDisconnected = false

	// Cancel reconnect timer if exists
	if c.publisherReconnectTimer != nil {
		c.publisherReconnectTimer.Stop()
		c.publisherReconnectTimer = nil
	}
	c.publisherMu.Unlock()

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
	c.subscribersMu.Lock()
	c.subscribers[subscriberID] = &Subscriber{ID: subscriberID, Conn: conn}

	// Cancel keep-alive timer if exists
	if c.keepAliveTimer != nil {
		c.keepAliveTimer.Stop()
		c.keepAliveTimer = nil
	}
	c.subscribersMu.Unlock()

	c.logger.Info("Subscriber connected", "subscriber_id", subscriberID, "total_subscribers", len(c.subscribers))

	// Send buffered Output messages to new subscriber
	// This allows new observers to see recent terminal output they missed
	bufferedOutput := c.getBufferedOutput()
	if len(bufferedOutput) > 0 {
		c.logger.Debug("Sending buffered output to new subscriber",
			"subscriber_id", subscriberID, "count", len(bufferedOutput))
		for _, data := range bufferedOutput {
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				c.logger.Warn("Failed to send buffered output to new subscriber",
					"subscriber_id", subscriberID, "error", err)
				break // Stop sending if connection has issues
			}
		}
	}

	// Notify new subscriber if publisher is currently disconnected
	if c.IsPublisherDisconnected() {
		if err := conn.WriteMessage(websocket.BinaryMessage, protocol.EncodeRunnerDisconnected()); err != nil {
			c.logger.Warn("Failed to send publisher disconnected status to new subscriber",
				"subscriber_id", subscriberID, "error", err)
		}
	}

	// Start forwarding from this subscriber to publisher
	go c.forwardSubscriberToPublisher(subscriberID)
}

// RemoveSubscriber removes a subscriber
func (c *TerminalChannel) RemoveSubscriber(subscriberID string) {
	c.subscribersMu.Lock()
	if subscriber, ok := c.subscribers[subscriberID]; ok {
		subscriber.Conn.Close()
		delete(c.subscribers, subscriberID)
	}
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
		c.lastSubscriberDisconnect = time.Now()
		c.keepAliveTimer = time.AfterFunc(c.config.KeepAliveDuration, func() {
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

// Broadcast sends data to all connected subscribers
func (c *TerminalChannel) Broadcast(data []byte) {
	c.subscribersMu.RLock()
	defer c.subscribersMu.RUnlock()

	subscriberCount := len(c.subscribers)
	c.logger.Debug("Broadcasting to subscribers", "data_len", len(data), "subscriber_count", subscriberCount)

	for _, subscriber := range c.subscribers {
		if err := subscriber.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			c.logger.Warn("Failed to send to subscriber", "subscriber_id", subscriber.ID, "error", err)
		} else {
			c.logger.Debug("Sent to subscriber", "subscriber_id", subscriber.ID, "data_len", len(data))
		}
	}
}

// bufferOutput adds an Output message to the ring buffer for new observers
func (c *TerminalChannel) bufferOutput(data []byte) {
	c.outputBufferMu.Lock()
	defer c.outputBufferMu.Unlock()

	dataLen := len(data)

	// Evict old messages if buffer is full (by count or size)
	for len(c.outputBuffer) >= c.config.OutputBufferCount || (c.outputBufferBytes+dataLen > c.config.OutputBufferSize && len(c.outputBuffer) > 0) {
		// Remove oldest message
		oldMsg := c.outputBuffer[0]
		c.outputBuffer = c.outputBuffer[1:]
		c.outputBufferBytes -= len(oldMsg)
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

// forwardPublisherToSubscribers forwards data from publisher to all subscribers
func (c *TerminalChannel) forwardPublisherToSubscribers() {
	c.logger.Debug("Starting forwardPublisherToSubscribers loop")
	for {
		c.publisherMu.RLock()
		conn := c.publisher
		c.publisherMu.RUnlock()

		if conn == nil {
			c.logger.Debug("Publisher conn is nil, exiting forward loop")
			break
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			c.logger.Info("Publisher disconnected", "error", err)
			c.handlePublisherDisconnect()
			break
		}

		c.logger.Debug("Received data from publisher", "data_len", len(data))

		// Buffer Output messages for new observers
		msg, _ := protocol.DecodeMessage(data)
		if msg != nil && msg.Type == protocol.MsgTypeOutput {
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

		// For input and image paste messages, check control permission
		if msg.Type == protocol.MsgTypeInput || msg.Type == protocol.MsgTypeImagePaste {
			if !c.CanInput(subscriberID) {
				c.logger.Debug("Input rejected, no control", "subscriber_id", subscriberID)
				continue
			}
		}

		// Handle ping/pong locally
		if msg.Type == protocol.MsgTypePing {
			if err := subscriber.Conn.WriteMessage(websocket.BinaryMessage, protocol.EncodePong()); err != nil {
				c.logger.Warn("Failed to send pong", "subscriber_id", subscriberID)
			}
			continue
		}

		// Forward to publisher
		c.publisherMu.RLock()
		if c.publisher != nil {
			if err := c.publisher.WriteMessage(websocket.BinaryMessage, data); err != nil {
				c.logger.Warn("Failed to forward to publisher", "error", err)
			}
		}
		c.publisherMu.RUnlock()
	}
}

// handlePublisherDisconnect handles publisher disconnect
// Instead of immediately closing the channel, wait for publisher to reconnect
func (c *TerminalChannel) handlePublisherDisconnect() {
	c.publisherMu.Lock()
	if c.publisher != nil {
		c.publisher.Close()
		c.publisher = nil
	}
	c.publisherDisconnected = true

	// Notify all subscribers that publisher has disconnected
	c.publisherMu.Unlock()
	c.Broadcast(protocol.EncodeRunnerDisconnected())
	c.publisherMu.Lock()

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
	}

	if response != nil {
		data, _ := protocol.EncodeControlRequest(response)
		c.subscribersMu.RLock()
		if subscriber, ok := c.subscribers[subscriberID]; ok {
			if err := subscriber.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				c.logger.Warn("Failed to send control response", "subscriber_id", subscriberID, "error", err)
			}
		}
		c.subscribersMu.RUnlock()
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
