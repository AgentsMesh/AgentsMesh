package channel

import (
	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

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
// Each goroutine is bound to a specific epoch. When SetPublisher replaces the publisher,
// it closes the old conn (causing ReadMessage to fail) and waits for this goroutine to
// exit before starting a new one.
func (c *TerminalChannel) forwardPublisherToSubscribers(epoch uint64) {
	defer c.publisherWg.Done()

	c.logger.Debug("Starting forwardPublisherToSubscribers loop", "epoch", epoch)

	c.publisherMu.RLock()
	conn := c.publisher
	currentEpoch := c.publisherEpoch
	c.publisherMu.RUnlock()

	// Epoch check: if epoch already changed, this goroutine is stale
	if conn == nil || currentEpoch != epoch {
		c.logger.Debug("Publisher epoch mismatch, exiting",
			"goroutine_epoch", epoch, "current_epoch", currentEpoch)
		return
	}

	conn.SetReadLimit(maxMessageSize)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.logger.Info("Publisher disconnected", "error", err, "epoch", epoch)
			c.handlePublisherDisconnect(conn, epoch)
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
		// Support both PTY Snapshot (0x01) and ACP Snapshot (0x0D).
		if msg != nil && (msg.Type == protocol.MsgTypeSnapshot || msg.Type == protocol.MsgTypeAcpSnapshot) {
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

		// For input, resize, and ACP command messages, check control permission
		if msg.Type == protocol.MsgTypeInput || msg.Type == protocol.MsgTypeResize || msg.Type == protocol.MsgTypeAcpCommand {
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
