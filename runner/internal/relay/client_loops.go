package relay

import (
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) readLoop() {
	c.logger.Debug("Read loop starting")

	// Capture one immutable outbound generation. A reconnect publishes a fresh
	// queue, so this loop can never consume or acknowledge the next socket's
	// frames.
	_, generation, doneCh, doneOnce, _ := c.snapshotOutbound()

	defer func() {
		// IMPORTANT: Call wg.Done() FIRST to ensure Stop() doesn't wait unnecessarily
		// This must happen before any callbacks that might block
		c.wg.Done()

		c.logger.Info("Read loop exited")

		// Closing done first unblocks a Flush that may be waiting for queue
		// capacity while holding outboundMu. The generation check then prevents
		// this old reader from deactivating a replacement connection.
		signalConnectionDone(doneCh, doneOnce)
		c.deactivateOutbound(generation)

		// Check if this is a graceful shutdown (Stop() called) or unexpected disconnect
		select {
		case <-c.stopCh:
			// Graceful shutdown - call onClose and don't reconnect
			c.fireOnClose()
		default:
			// Flap detection: if the connection died quickly, increment the
			// reconnect counter so reconnectLoop applies increasing backoff.
			// This prevents 500ms tight-loop reconnects when the relay keeps
			// closing us immediately (e.g., no subscriber waiting).
			connAt := c.connectedAt.Load()
			connDuration := time.Since(time.UnixMilli(connAt))
			if connDuration < minStableConnected {
				count := c.reconnectCount.Add(1)
				c.logger.Warn("Connection was short-lived, increasing reconnect backoff",
					"duration", connDuration, "reconnect_count", count)
			} else {
				c.reconnectCount.Store(0)
			}

			// Atomically check stopped and add reconnectLoop to wg under lock.
			// This ensures Stop() will wait for reconnectLoop to exit.
			c.wgMu.Lock()
			if c.stopped.Load() {
				c.wgMu.Unlock()
				c.fireOnClose()
				return
			}
			if !c.reconnecting.Swap(true) {
				c.wg.Add(1)
				c.wgMu.Unlock()
				go c.reconnectLoop()
			} else {
				c.wgMu.Unlock()
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

func (c *Client) handleMessage(data []byte) {
	msg, err := DecodeMessage(data)
	if err != nil {
		c.logger.Error("Failed to decode message", "error", err)
		return
	}

	switch msg.Type {
	case MsgTypePing:
		c.SendPong()
	case MsgTypePong:
	default:
		c.handlersMu.RLock()
		h := c.handlers[msg.Type]
		c.handlersMu.RUnlock()
		if h != nil {
			h(msg.Payload)
		}
	}
}
