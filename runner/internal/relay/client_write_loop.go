package relay

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	c.writeLoopWithPing(ticker.C)
}

func (c *Client) writeLoopWithPing(ping <-chan time.Time) {
	c.logger.Debug("Write loop starting")
	defer c.wg.Done()
	defer c.logger.Info("Write loop exited")

	// Signal reconnectLoop that writeLoop has fully exited.
	queue, generation, doneCh, doneOnce, exitCh := c.snapshotOutbound()
	defer c.finishOutboundWriter(queue, generation, exitCh)

	for {
		select {
		case <-c.stopCh:
			return

		case <-doneCh:
			// Connection is done (readLoop exited), stop writeLoop
			return

		case item := <-queue:
			if item.generation != generation {
				if item.flush != nil {
					item.flush <- errRelayGenerationEnded
				}
				continue
			}
			if item.flush != nil {
				item.flush <- nil
				continue
			}
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				signalConnectionDone(doneCh, doneOnce)
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.BinaryMessage, item.data); err != nil {
				c.logger.Error("Write error", "error", err)
				signalConnectionDone(doneCh, doneOnce)
				_ = conn.Close()
				return
			}

		case <-ping:
			c.connMu.RLock()
			conn := c.conn
			c.connMu.RUnlock()

			if conn == nil {
				signalConnectionDone(doneCh, doneOnce)
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error("Ping error", "error", err)
				signalConnectionDone(doneCh, doneOnce)
				_ = conn.Close()
				return
			}
		}
	}
}

func signalConnectionDone(ch chan struct{}, once *sync.Once) {
	if ch == nil || once == nil {
		return
	}
	once.Do(func() { close(ch) })
}

func (c *Client) finishOutboundWriter(queue chan outboundItem, generation uint64, exitCh chan struct{}) {
	c.deactivateOutbound(generation)
	discardOutbound(queue, errRelayGenerationEnded)
	if exitCh == nil {
		return
	}
	select {
	case <-exitCh:
	default:
		close(exitCh)
	}
}
