package relay

import (
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) readLoop() {
	c.logger.Debug("Read loop starting")
	defer c.wg.Done()
	defer func() {
		c.connected.Store(false)
		c.logger.Info("Read loop exited")

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
	c.logger.Debug("Write loop starting")
	defer c.wg.Done()
	defer c.logger.Info("Write loop exited")

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
