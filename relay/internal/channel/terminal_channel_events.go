package channel

import (
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// handlePublisherDisconnect handles publisher disconnect with double identity check
// (pointer + epoch). If SetPublisher already replaced the publisher, this is a no-op.
func (c *TerminalChannel) handlePublisherDisconnect(disconnectedConn *websocket.Conn, epoch uint64) {
	c.publisherMu.Lock()

	// Double identity check: pointer AND epoch must both match.
	// This eliminates the race where an old goroutine's disconnect handler
	// runs after SetPublisher has already installed a new connection.
	if c.publisher != disconnectedConn || c.publisherEpoch != epoch {
		c.publisherMu.Unlock()
		return
	}

	// Save reference, nil out under lock, then close outside lock
	// to avoid blocking other goroutines while Close() performs I/O.
	conn := c.publisher
	c.publisher = nil
	c.publisherDisconnected = true

	c.logger.Info("Publisher disconnected, waiting for reconnection",
		"timeout", c.config.PublisherReconnectTimeout, "epoch", epoch)

	// Start reconnect timer
	c.publisherReconnectTimer = time.AfterFunc(c.config.PublisherReconnectTimeout, func() {
		// Acquire publisherReplaceMu first to serialize with SetPublisher/Close.
		// This prevents the TOCTOU race where timer reads stillDisconnected=true,
		// then SetPublisher reconnects, then timer proceeds to Close() the new connection.
		c.publisherReplaceMu.Lock()

		c.publisherMu.Lock()
		stillDisconnected := c.publisherDisconnected
		c.publisherMu.Unlock()

		if !stillDisconnected {
			c.publisherReplaceMu.Unlock()
			return
		}

		// Still disconnected — close the channel.
		// We already hold publisherReplaceMu, so call closeInternal directly
		// to avoid re-acquiring it (which would deadlock).
		c.logger.Info("Publisher reconnect timeout, closing channel")
		c.closeInternal()
		c.publisherReplaceMu.Unlock()
	})
	c.publisherMu.Unlock()

	// Close connection after releasing lock
	_ = conn.Close()

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
