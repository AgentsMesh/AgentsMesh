package relay

import (
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) readLoop() {
	c.logger.Debug("Read loop starting")

	// Capture connDoneCh at loop start so that even if reconnectLoop replaces
	// c.connDoneCh with a new channel, we only close the one belonging to
	// this connection iteration.
	doneCh := c.connDoneCh

	defer func() {
		// IMPORTANT: Call wg.Done() FIRST to ensure Stop() doesn't wait unnecessarily
		// This must happen before any callbacks that might block
		c.wg.Done()

		c.connected.Store(false)
		c.logger.Info("Read loop exited")

		// Signal writeLoop that this connection is done
		// Safe to close multiple times via select
		select {
		case <-doneCh:
			// Already closed
		default:
			close(doneCh)
		}

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
	c.logger.Debug("Write loop starting")
	defer c.wg.Done()
	defer c.logger.Info("Write loop exited")

	// Capture connDoneCh at loop start — matches readLoop pattern.
	// If reconnectLoop replaces c.connDoneCh, we only listen to the one
	// belonging to this connection iteration.
	doneCh := c.connDoneCh

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return

		case <-doneCh:
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
	case MsgTypePing:
		c.SendPong()
	case MsgTypePong:
		// Received pong, connection is alive
	default:
		c.handlersMu.RLock()
		h := c.handlers[msg.Type]
		c.handlersMu.RUnlock()
		if h != nil {
			h(msg.Payload)
		}
	}
}
