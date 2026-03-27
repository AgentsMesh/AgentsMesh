// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// sendWithTimeout sends a message with a timeout to prevent blocking forever.
// This is used for critical messages like initialization where we can't afford to block.
// On timeout, clears the stream reference so the blocked Send goroutine will eventually
// return when the underlying connection is closed (consistent with sendAndRecord behavior).
func (c *GRPCConnection) sendWithTimeout(msg *runnerv1.RunnerMessage, timeout time.Duration) error {
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()

	if stream == nil {
		return fmt.Errorf("stream not connected")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- stream.Send(msg)
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		// Close the stream to unblock the goroutine stuck on Send().
		// Just setting stream=nil leaves the goroutine alive until reconnect.
		stream.CloseSend()
		c.mu.Lock()
		c.stream = nil
		c.mu.Unlock()
		return fmt.Errorf("send timed out after %v", timeout)
	}
}

// performInitialization performs the three-phase initialization handshake.
func (c *GRPCConnection) performInitialization(ctx context.Context) error {
	logger.GRPC().Debug("Starting initialization handshake...")

	// Drain any stale result from a previous connection's initResultCh
	// to prevent reading an outdated InitializeResult.
	select {
	case <-c.initResultCh:
	default:
	}

	// Use a shorter timeout for initialization messages (5s)
	// This ensures we fail fast if stream.Send() is blocking
	const initSendTimeout = 5 * time.Second

	// Phase 1: Send initialize request
	hostname, _ := os.Hostname()
	initReq := &runnerv1.InitializeRequest{
		ProtocolVersion: GRPCProtocolVersion,
		RunnerInfo: &runnerv1.RunnerInfo{
			Version:  c.runnerVersion,
			NodeId:   c.nodeID,
			McpPort:  int32(c.mcpPort),
			Os:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Hostname: hostname,
		},
	}

	// Send initialize request via stream (with timeout)
	msg := &runnerv1.RunnerMessage{
		Payload:   &runnerv1.RunnerMessage_Initialize{Initialize: initReq},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.sendWithTimeout(msg, initSendTimeout); err != nil {
		return fmt.Errorf("failed to send initialize: %w", err)
	}
	logger.GRPC().Debug("Sent initialize request", "version", c.runnerVersion, "mcp_port", c.mcpPort)

	// Phase 2: Wait for initialize_result
	select {
	case result := <-c.initResultCh:
		logger.GRPC().Debug("Received initialize_result",
			"server_version", result.ServerInfo.Version,
			"agents", len(result.Agents))

		// Phase 3: Check available agents (with version detection) and send initialized
		availableAgents, agentVersions := c.agentProbe.ProbeAll(result.Agents)
		c.mu.Lock()
		c.availableAgents = availableAgents
		c.mu.Unlock()

		// Send initialized confirmation via stream (with timeout)
		// Includes both legacy available_agents (slug list) and new agent_versions (with version info)
		confirmMsg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialized{
				Initialized: &runnerv1.InitializedConfirm{
					AvailableAgents: availableAgents,
					AgentVersions:   agentVersions,
				},
			},
			Timestamp: time.Now().UnixMilli(),
		}
		if err := c.sendWithTimeout(confirmMsg, initSendTimeout); err != nil {
			return fmt.Errorf("failed to send initialized: %w", err)
		}
		logger.GRPC().Debug("Sent initialized", "available_agents", availableAgents, "agent_versions", len(agentVersions))

		c.mu.Lock()
		c.initialized = true
		c.mu.Unlock()

		logger.GRPC().Info("Initialization completed successfully")
		return nil

	case <-time.After(c.initTimeout):
		return fmt.Errorf("timeout waiting for initialize_result after %v", c.initTimeout)

	case <-ctx.Done():
		return fmt.Errorf("context cancelled during initialization")

	case <-c.stopCh:
		return fmt.Errorf("connection stopped during initialization")
	}
}
