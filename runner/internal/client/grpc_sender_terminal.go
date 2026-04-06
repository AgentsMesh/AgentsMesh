package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// NOTE: SendTerminalOutput removed - output is exclusively streamed via Relay

// SendAgentStatus sends an agent status change event to the server (terminal message).
func (c *GRPCConnection) SendAgentStatus(podKey string, status string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AgentStatus{
			AgentStatus: &runnerv1.AgentStatusEvent{
				PodKey: podKey,
				Status: status,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendTerminal(msg)
}

// SendOSCNotification sends an OSC notification event to the server (control message).
// This is triggered by OSC 777 (iTerm2/Kitty) or OSC 9 (ConEmu/Windows Terminal) sequences.
// Uses controlCh for high priority delivery (not affected by terminal output throttling).
func (c *GRPCConnection) SendOSCNotification(podKey, title, body string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_OscNotification{
			OscNotification: &runnerv1.OSCNotificationEvent{
				PodKey:    podKey,
				Title:     title,
				Body:      body,
				Timestamp: time.Now().UnixMilli(),
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendOSCTitle sends an OSC title change event to the server (control message).
// This is triggered by OSC 0/2 sequences for window/tab title changes.
func (c *GRPCConnection) SendOSCTitle(podKey, title string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_OscTitle{
			OscTitle: &runnerv1.OSCTitleEvent{
				PodKey: podKey,
				Title:  title,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// QueueLength returns the current terminal send queue length.
func (c *GRPCConnection) QueueLength() int {
	return len(c.terminalCh)
}

// QueueCapacity returns the terminal send queue capacity.
func (c *GRPCConnection) QueueCapacity() int {
	return cap(c.terminalCh)
}

// QueueUsage returns the terminal queue usage ratio (0.0 to 1.0).
// Used for monitoring queue pressure.
func (c *GRPCConnection) QueueUsage() float64 {
	return float64(len(c.terminalCh)) / float64(cap(c.terminalCh))
}

// drainTerminalQueue clears all pending messages in the terminal queue.
// Called before reconnection to discard stale terminal output.
// TUI frames are expendable - old frames are irrelevant after reconnection.
func (c *GRPCConnection) drainTerminalQueue() {
	drained := 0
	for {
		select {
		case <-c.terminalCh:
			drained++
		default:
			if drained > 0 {
				logger.GRPC().Info("Drained stale terminal queue before reconnection",
					"messages_dropped", drained)
			}
			return
		}
	}
}
