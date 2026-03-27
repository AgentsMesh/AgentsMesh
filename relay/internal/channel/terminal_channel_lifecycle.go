package channel

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

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
		_ = conn.Close()
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
	_ = subscriber.Conn.Close()
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
		c.handleLastSubscriberGone()
	}
}

// handleLastSubscriberGone starts the keep-alive timer when no subscribers remain.
func (c *TerminalChannel) handleLastSubscriberGone() {
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

// SubscriberCount returns the number of connected subscribers
func (c *TerminalChannel) SubscriberCount() int {
	c.subscribersMu.RLock()
	defer c.subscribersMu.RUnlock()
	return len(c.subscribers)
}

// Close closes the channel and all connections.
// Safe for concurrent callers — only the first call performs cleanup.
func (c *TerminalChannel) Close() {
	c.publisherReplaceMu.Lock()
	c.closeInternal()
	c.publisherReplaceMu.Unlock()
}

// closeInternal performs the actual close logic.
// MUST be called with publisherReplaceMu held.
func (c *TerminalChannel) closeInternal() {
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
		_ = c.publisher.Close()
		c.publisher = nil
	}
	c.publisherMu.Unlock()

	// Wait for publisher forwarding goroutine to exit.
	// conn.Close() above triggers ReadMessage error, causing the goroutine to return.
	c.publisherWg.Wait()

	// Close all subscriber connections
	c.subscribersMu.Lock()
	for _, subscriber := range c.subscribers {
		_ = subscriber.Conn.Close()
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
