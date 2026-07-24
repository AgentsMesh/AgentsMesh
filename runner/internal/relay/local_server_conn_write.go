package relay

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

// enqueue adds an immutable, already-encoded frame without blocking its
// caller. An overloaded connection is isolated and closed; other connections
// for the pod continue independently.
func (c *localConn) enqueue(frame []byte) bool {
	frameBytes := len(frame)
	c.queueMu.Lock()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.queueMu.Unlock()
		return false
	}
	if c.queuedItems >= localWriteQueueItems || frameBytes > localWriteQueueByteBudget-c.queuedBytes {
		c.mu.Unlock()
		c.queueMu.Unlock()
		c.fail(errLocalWriteQueueOverflow)
		return false
	}
	select {
	case c.queue <- localWriteItem{frame: frame}:
		c.queuedBytes += frameBytes
		c.queuedItems++
		c.mu.Unlock()
		c.queueMu.Unlock()
		return true
	default:
		c.mu.Unlock()
		c.queueMu.Unlock()
		c.fail(errLocalWriteQueueOverflow)
		return false
	}
}

// flush inserts a marker behind every frame accepted by enqueue before this
// call and waits for the single socket writer to reach it.
func (c *localConn) flush(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ack := make(chan error, 1)
	marker := localWriteItem{flush: ack}

	// queueMu gives concurrent Send and Flush calls a single FIFO insertion
	// order without holding the lifecycle/accounting mutex while waiting for a
	// queue slot.
	c.queueMu.Lock()
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		c.queueMu.Unlock()
		return errLocalConnClosed
	}
	select {
	case c.queue <- marker:
		c.queueMu.Unlock()
	case <-ctx.Done():
		c.queueMu.Unlock()
		return ctx.Err()
	case <-c.done:
		c.queueMu.Unlock()
		return errLocalConnClosed
	}

	select {
	case err := <-ack:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-c.done:
		return preferLocalFlushAck(ack, errLocalConnClosed)
	}
}

func preferLocalFlushAck(ack <-chan error, fallback error) error {
	select {
	case err := <-ack:
		return err
	default:
		return fallback
	}
}

func (c *localConn) writeLoop() {
	defer close(c.writerDone)
	defer c.discardQueued()

	for {
		select {
		case <-c.done:
			return
		case item := <-c.queue:
			if c.isClosed() {
				c.releaseItem(item, errLocalConnClosed)
				return
			}
			if item.flush != nil {
				item.flush <- nil
				continue
			}
			_ = c.socket.SetWriteDeadline(time.Now().Add(localWriteTimeout))
			err := c.socket.WriteMessage(websocket.BinaryMessage, item.frame)
			c.releaseItem(item, err)
			if err != nil {
				c.fail(err)
				return
			}
		}
	}
}

func (c *localConn) releaseItem(item localWriteItem, flushErr error) {
	if item.flush != nil {
		item.flush <- flushErr
		return
	}
	c.mu.Lock()
	c.queuedBytes -= len(item.frame)
	c.queuedItems--
	c.mu.Unlock()
}

func (c *localConn) discardQueued() {
	for {
		select {
		case item := <-c.queue:
			c.releaseItem(item, errLocalConnClosed)
		default:
			return
		}
	}
}

func (c *localConn) queueUsage() (bytes, items int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.queuedBytes, c.queuedItems
}
