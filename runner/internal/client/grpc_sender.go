// Package client provides gRPC connection management for Runner.
package client

import (
	"fmt"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// sendControl queues a control message (high priority).
// Control messages include: heartbeat, pod_created, pod_terminated, pty_resized, error.
// These messages should never be blocked by terminal output.
// Returns error if connection is closed, stopped, or channel is full.
func (c *GRPCConnection) sendControl(msg *runnerv1.RunnerMessage) error {
	c.mu.Lock()
	if c.stream == nil {
		c.mu.Unlock()
		return fmt.Errorf("stream not connected")
	}
	c.mu.Unlock()

	select {
	case c.controlCh <- msg:
		return nil
	case <-c.stopCh:
		return fmt.Errorf("connection stopped")
	default:
		logger.GRPC().Warn("Control buffer full, message dropped",
			"queue_len", len(c.controlCh))
		return fmt.Errorf("control buffer full")
	}
}

// sendTerminal queues a terminal message (low priority).
// Terminal messages include: agent_status.
// NOTE: terminal_output removed - terminal output is exclusively streamed via Relay.
// These messages are dropped silently if buffer is full.
// Returns nil even when dropped to avoid blocking callers.
//
// IMPORTANT: Messages are rejected before initialization completes.
// This prevents queue buildup during reconnection handshake, which could
// cause gRPC flow control to block the initialize_result response.
func (c *GRPCConnection) sendTerminal(msg *runnerv1.RunnerMessage) error {
	c.mu.Lock()
	stream := c.stream
	initialized := c.initialized
	c.mu.Unlock()

	// Reject messages before initialization completes
	// During reconnection, old Pods may still produce output, but sending it
	// before handshake completes can block the gRPC stream and cause deadlock
	if !initialized {
		logger.Terminal().Debug("sendTerminal: not initialized, dropping message")
		return nil // Silent drop, not an error
	}

	if stream == nil {
		logger.Terminal().Debug("sendTerminal: stream not connected")
		return fmt.Errorf("stream not connected")
	}

	select {
	case c.terminalCh <- msg:
		logger.Terminal().Debug("sendTerminal: message queued",
			"queue_len", len(c.terminalCh))
		return nil
	case <-c.stopCh:
		logger.Terminal().Debug("sendTerminal: connection stopped")
		return fmt.Errorf("connection stopped")
	default:
		// TUI frames are expendable - drop silently
		logger.GRPC().Debug("Terminal output dropped (queue full)",
			"queue_usage", c.QueueUsage())
		return nil
	}
}
