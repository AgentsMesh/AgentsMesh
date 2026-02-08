// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// writeLoop sends messages to the gRPC stream with priority scheduling.
// Control messages (heartbeat, pod events) have higher priority than terminal output.
// This is the ONLY goroutine that calls stream.Send() to ensure thread-safety.
// Includes stuck detection: triggers reconnect if no successful send for 30 seconds.
func (c *GRPCConnection) writeLoop(ctx context.Context, done <-chan struct{}) {
	log := logger.GRPC()
	log.Info("Write loop starting")
	defer log.Info("Write loop exited")

	stuckTicker := time.NewTicker(10 * time.Second)
	defer stuckTicker.Stop()

	// Initialize last send time
	c.lastSendTime.Store(time.Now().UnixNano())

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return

		case <-stuckTicker.C:
			// Stuck detection: if no successful send for 30 seconds, trigger reconnect
			lastSend := time.Unix(0, c.lastSendTime.Load())
			if time.Since(lastSend) > 30*time.Second {
				log.Error("WriteLoop stuck for 30s, triggering reconnect")
				c.triggerReconnect()
				return
			}

		case msg := <-c.controlCh:
			// Control messages have highest priority
			c.sendAndRecord(msg)

		default:
			// No control messages pending - use nested select for priority
			select {
			case <-c.stopCh:
				return
			case <-done:
				return
			case <-ctx.Done():
				return
			case msg := <-c.controlCh:
				// Double-check for control messages (priority)
				c.sendAndRecord(msg)
			case msg := <-c.terminalCh:
				// Process terminal messages when no control messages pending
				logger.GRPCTrace().Trace("writeLoop: sending terminal message",
					"queue_len", len(c.terminalCh))
				c.sendAndRecord(msg)
			}
		}
	}
}

// sendAndRecord sends a message with a hard timeout to prevent writeLoop from blocking forever.
// If stream.Send() doesn't complete within sendTimeout, the message is abandoned and we continue.
// This prevents a slow/stuck stream.Send() from blocking all message processing.
//
// Key insight: gRPC stream.Send() can block indefinitely due to flow control.
// We cannot cancel it, but we can stop waiting and move on.
func (c *GRPCConnection) sendAndRecord(msg *runnerv1.RunnerMessage) {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		logger.GRPC().Warn("sendAndRecord: stream is nil, dropping message")
		return
	}

	// Use a goroutine with timeout to prevent blocking forever
	// The send operation runs in a goroutine, and we wait with a timeout
	const sendTimeout = 5 * time.Second

	type sendResult struct {
		err     error
		elapsed time.Duration
	}

	resultCh := make(chan sendResult, 1)
	start := time.Now()

	go func() {
		err := stream.Send(msg)
		resultCh <- sendResult{err: err, elapsed: time.Since(start)}
	}()

	select {
	case result := <-resultCh:
		// Send completed (success or failure)
		if result.err != nil {
			logger.GRPC().Error("Failed to send message", "error", result.err, "elapsed", result.elapsed)
			return
		}

		// Log slow sends for diagnosis
		if result.elapsed > 100*time.Millisecond {
			logger.GRPC().Warn("Slow stream.Send()", "elapsed", result.elapsed,
				"terminal_queue", len(c.terminalCh))
		}

		// Update last successful send time
		c.lastSendTime.Store(time.Now().UnixNano())

	case <-time.After(sendTimeout):
		// Send timed out - the goroutine is still running but we move on
		// This prevents writeLoop from being blocked forever
		logger.GRPC().Error("stream.Send() timed out, abandoning message and triggering reconnect",
			"timeout", sendTimeout, "terminal_queue", len(c.terminalCh))

		// Trigger reconnect to recover from degraded connection
		// The stuck goroutine will eventually complete or error when stream is closed
		c.triggerReconnect()
	}
}

// heartbeatLoop sends periodic heartbeats.
func (c *GRPCConnection) heartbeatLoop(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	c.sendHeartbeat()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends a heartbeat message (control message - never blocked by terminal output).
func (c *GRPCConnection) sendHeartbeat() {
	var pods []*runnerv1.PodInfo
	var relayConnections []*runnerv1.RelayConnectionInfo

	if c.handler != nil {
		// Convert from internal PodInfo to proto PodInfo
		internalPods := c.handler.OnListPods()
		for _, p := range internalPods {
			pods = append(pods, &runnerv1.PodInfo{
				PodKey:      p.PodKey,
				Status:      p.Status,
				AgentStatus: p.ClaudeStatus,
			})
		}

		// Convert from internal RelayConnectionInfo to proto RelayConnectionInfo
		internalRelayConns := c.handler.OnListRelayConnections()
		for _, rc := range internalRelayConns {
			relayConnections = append(relayConnections, &runnerv1.RelayConnectionInfo{
				PodKey:      rc.PodKey,
				RelayUrl:    rc.RelayURL,
				Connected:   rc.Connected,
				ConnectedAt: rc.ConnectedAt,
			})
		}
	}

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{
				NodeId:           c.nodeID,
				Pods:             pods,
				RelayConnections: relayConnections,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	logger.GRPC().Debug("Sending heartbeat", "pods", len(pods), "relay_connections", len(relayConnections))

	if err := c.sendControl(msg); err != nil {
		logger.GRPC().Error("Failed to send heartbeat", "error", err)
	}
}
