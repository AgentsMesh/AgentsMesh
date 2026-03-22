package relay

import "fmt"

// Send sends a message with the given type and payload via the relay.
func (c *Client) Send(msgType byte, payload []byte) error {
	return c.send(EncodeMessage(msgType, payload))
}

// SendPong sends a pong response (internal, used by handleMessage).
func (c *Client) SendPong() error {
	return c.send(EncodePong())
}

func (c *Client) send(data []byte) error {
	if !c.connected.Load() {
		return fmt.Errorf("not connected")
	}

	select {
	case c.sendCh <- data:
		return nil
	default:
		// Channel full, drop the message
		c.logger.Warn("Send channel full, dropping message")
		return fmt.Errorf("send buffer full")
	}
}
