package relay

import (
	"fmt"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/terminal/vt"
)

// SendSnapshot sends a terminal snapshot to the relay
func (c *Client) SendSnapshot(snapshot *vt.TerminalSnapshot) error {
	data, err := EncodeSnapshot(snapshot)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	return c.send(data)
}

// SendOutput sends terminal output to the relay
func (c *Client) SendOutput(data []byte) error {
	return c.send(EncodeOutput(data))
}

// SendPong sends a pong response
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
