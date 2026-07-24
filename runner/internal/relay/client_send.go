package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	errRelayFlushDisconnected = errors.New("relay connection ended before output flush completed")
	errRelayClientStopped     = errors.New("relay client stopped before output flush completed")
	errRelayGenerationEnded   = errors.New("relay outbound generation ended before output flush completed")
)

// Send sends a message with the given type and payload via the relay.
func (c *Client) Send(msgType byte, payload []byte) error {
	return c.send(EncodeMessage(msgType, payload))
}

// SendPong sends a pong response (internal, used by handleMessage).
func (c *Client) SendPong() error {
	return c.send(EncodePong())
}

func (c *Client) send(data []byte) error {
	c.outboundMu.Lock()
	defer c.outboundMu.Unlock()
	if !c.outboundAccepting || !c.connected.Load() {
		return fmt.Errorf("not connected")
	}
	select {
	case <-c.stopCh:
		return fmt.Errorf("not connected")
	case <-c.outboundDoneCh:
		return fmt.Errorf("not connected")
	case <-c.outboundWriteExit:
		return fmt.Errorf("not connected")
	default:
	}
	item := outboundItem{data: data, generation: c.outboundGeneration}
	select {
	case c.sendCh <- item:
		return nil
	default:
		// Channel full, drop the message
		c.logger.Warn("Send channel full, dropping message")
		return fmt.Errorf("send buffer full")
	}
}

// Flush inserts a marker in the outbound FIFO and waits until the connection's
// single writer reaches it. Reaching the marker proves every previously
// accepted frame has completed WriteMessage, not merely entered sendCh.
func (c *Client) Flush(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.outboundMu.Lock()
	if !c.outboundAccepting || !c.connected.Load() {
		c.outboundMu.Unlock()
		return fmt.Errorf("not connected")
	}
	doneCh := c.outboundDoneCh
	writeExitCh := c.outboundWriteExit
	queue := c.sendCh
	generation := c.outboundGeneration
	ack := make(chan error, 1)
	marker := outboundItem{flush: ack, generation: generation}
	select {
	case <-c.stopCh:
		c.outboundMu.Unlock()
		return errRelayClientStopped
	case <-doneCh:
		c.outboundMu.Unlock()
		return errRelayFlushDisconnected
	case <-writeExitCh:
		c.outboundMu.Unlock()
		return errRelayFlushDisconnected
	default:
	}
	select {
	case queue <- marker:
		c.outboundMu.Unlock()
	case <-ctx.Done():
		c.outboundMu.Unlock()
		return ctx.Err()
	case <-c.stopCh:
		c.outboundMu.Unlock()
		return errRelayClientStopped
	case <-doneCh:
		c.outboundMu.Unlock()
		return errRelayFlushDisconnected
	case <-writeExitCh:
		c.outboundMu.Unlock()
		return errRelayFlushDisconnected
	}

	select {
	case err := <-ack:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stopCh:
		return preferFlushAck(ack, errRelayClientStopped)
	case <-doneCh:
		return preferFlushAck(ack, errRelayFlushDisconnected)
	case <-writeExitCh:
		return preferFlushAck(ack, errRelayFlushDisconnected)
	}
}

func (c *Client) activateOutbound(
	queue chan outboundItem,
	doneCh chan struct{},
	writeExitCh chan struct{},
) bool {
	c.outboundMu.Lock()
	defer c.outboundMu.Unlock()
	if c.stopped.Load() {
		c.outboundAccepting = false
		c.connected.Store(false)
		return false
	}
	c.outboundGeneration++
	c.sendCh = queue
	c.outboundDoneCh = doneCh
	c.outboundDoneOnce = &sync.Once{}
	c.outboundWriteExit = writeExitCh
	c.outboundAccepting = true
	c.connected.Store(true)
	c.connectedAt.Store(time.Now().UnixMilli())
	return true
}

// deactivateOutbound stops acceptance for generation. generation==0 is the
// unconditional permanent-stop form.
func (c *Client) deactivateOutbound(generation uint64) {
	c.outboundMu.Lock()
	if generation == 0 || generation == c.outboundGeneration {
		c.outboundAccepting = false
		c.connected.Store(false)
	}
	c.outboundMu.Unlock()
}

func (c *Client) snapshotOutbound() (
	queue chan outboundItem,
	generation uint64,
	doneCh chan struct{},
	doneOnce *sync.Once,
	writeExitCh chan struct{},
) {
	c.outboundMu.Lock()
	defer c.outboundMu.Unlock()
	return c.sendCh, c.outboundGeneration, c.outboundDoneCh, c.outboundDoneOnce, c.outboundWriteExit
}

func discardOutbound(queue chan outboundItem, err error) {
	if queue == nil {
		return
	}
	for {
		select {
		case item := <-queue:
			if item.flush != nil {
				item.flush <- err
			}
		default:
			return
		}
	}
}

func preferFlushAck(ack <-chan error, fallback error) error {
	select {
	case err := <-ack:
		return err
	default:
		return fallback
	}
}
