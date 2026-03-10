// Package client provides gRPC connection management for Runner.
package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// handleTerminalInput handles terminal_input command from server.
func (c *GRPCConnection) handleTerminalInput(cmd *runnerv1.TerminalInputCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalInputRequest{
		PodKey: cmd.PodKey,
		Data:   cmd.Data, // gRPC uses native bytes, no encoding needed
	}
	if err := c.handler.OnTerminalInput(req); err != nil {
		logger.GRPC().Error("Failed to send terminal input to pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalResize handles terminal_resize command from server.
func (c *GRPCConnection) handleTerminalResize(cmd *runnerv1.TerminalResizeCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalResizeRequest{
		PodKey: cmd.PodKey,
		Cols:   uint16(cmd.Cols),
		Rows:   uint16(cmd.Rows),
	}
	if err := c.handler.OnTerminalResize(req); err != nil {
		logger.GRPC().Error("Failed to resize terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalRedraw handles terminal_redraw command from server.
// Uses resize +1/-1 trick to trigger terminal redraw for state recovery.
func (c *GRPCConnection) handleTerminalRedraw(cmd *runnerv1.TerminalRedrawCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalRedrawRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnTerminalRedraw(req); err != nil {
		logger.GRPC().Error("Failed to redraw terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSendPrompt handles send_prompt command from server.
func (c *GRPCConnection) handleSendPrompt(cmd *runnerv1.SendPromptCommand) {
	logger.GRPC().Debug("Received send_prompt", "pod_key", cmd.PodKey)
	// TODO: Implement prompt sending when handler supports it
}

// handleSubscribeTerminal handles subscribe_terminal command from server.
// This notifies the Runner that a browser wants to observe the terminal via Relay.
// Channel is identified by PodKey (not session ID).
func (c *GRPCConnection) handleSubscribeTerminal(cmd *runnerv1.SubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received subscribe_terminal", "pod_key", cmd.PodKey, "relay_url", cmd.RelayUrl)
	if c.handler == nil {
		log.Warn("No handler set, ignoring subscribe_terminal")
		return
	}

	req := SubscribeTerminalRequest{
		PodKey:          cmd.PodKey,
		RelayURL:        cmd.RelayUrl,
		RunnerToken:     cmd.RunnerToken,
		IncludeSnapshot: cmd.IncludeSnapshot,
		SnapshotHistory: cmd.SnapshotHistory,
	}
	if err := c.handler.OnSubscribeTerminal(req); err != nil {
		log.Error("Failed to subscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleUnsubscribeTerminal handles unsubscribe_terminal command from server.
// This notifies the Runner that all browsers have disconnected.
func (c *GRPCConnection) handleUnsubscribeTerminal(cmd *runnerv1.UnsubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received unsubscribe_terminal", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring unsubscribe_terminal")
		return
	}

	req := UnsubscribeTerminalRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnUnsubscribeTerminal(req); err != nil {
		log.Error("Failed to unsubscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleQuerySandboxes handles query_sandboxes command from server.
// Returns sandbox status for specified pod keys.
func (c *GRPCConnection) handleQuerySandboxes(cmd *runnerv1.QuerySandboxesCommand) {
	log := logger.GRPC()
	log.Info("Received query_sandboxes", "request_id", cmd.RequestId, "queries", len(cmd.Queries))
	if c.handler == nil {
		log.Warn("No handler set, ignoring query_sandboxes")
		return
	}

	req := QuerySandboxesRequest{
		RequestID: cmd.RequestId,
		Queries:   cmd.Queries,
	}
	if err := c.handler.OnQuerySandboxes(req); err != nil {
		log.Error("Failed to query sandboxes", "request_id", cmd.RequestId, "error", err)
	}
}

// handleCreateAutopilot handles create_autopilot command from server.
func (c *GRPCConnection) handleCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) {
	log := logger.GRPC()
	log.Info("Received create_autopilot", "autopilot_key", cmd.AutopilotKey, "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring create_autopilot")
		return
	}

	if err := c.handler.OnCreateAutopilot(cmd); err != nil {
		log.Error("Failed to create Autopilot", "autopilot_key", cmd.AutopilotKey, "error", err)
	}
}

// handleAutopilotControl handles autopilot_control command from server.
func (c *GRPCConnection) handleAutopilotControl(cmd *runnerv1.AutopilotControlCommand) {
	log := logger.GRPC()
	log.Info("Received autopilot_control", "autopilot_key", cmd.AutopilotKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring autopilot_control")
		return
	}

	if err := c.handler.OnAutopilotControl(cmd); err != nil {
		log.Error("Failed to handle Autopilot control", "autopilot_key", cmd.AutopilotKey, "error", err)
	}
}

// handleMcpResponse handles MCP response from server.
// Routes the response to RPCClient for request-response correlation.
func (c *GRPCConnection) handleMcpResponse(resp *runnerv1.McpResponse) {
	if c.rpcClient == nil {
		logger.GRPC().Warn("Received MCP response but no RPCClient set", "request_id", resp.RequestId)
		return
	}
	c.rpcClient.HandleResponse(resp)
}

// handleHeartbeatAck handles the server's acknowledgment of our heartbeat.
// Confirms the upstream path (Runner -> Backend) is alive by resetting
// the heartbeat monitor's missed-ack counter.
func (c *GRPCConnection) handleHeartbeatAck(ack *runnerv1.HeartbeatAck) {
	if c.heartbeatMonitor != nil {
		c.heartbeatMonitor.OnAck()
	}
	rtt := time.Now().UnixMilli() - ack.HeartbeatTimestamp
	logger.GRPCTrace().Trace("Heartbeat ack received", "rtt_ms", rtt)
}

// handlePing handles downstream ping from server - immediately replies with pong.
// This is a lightweight synchronous operation to maintain ordering with other control messages.
func (c *GRPCConnection) handlePing(ping *runnerv1.PingCommand) {
	if err := c.sendControl(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Pong{
			Pong: &runnerv1.PongEvent{
				PingTimestamp: ping.Timestamp,
			},
		},
	}); err != nil {
		logger.GRPC().Warn("Failed to send pong response", "error", err)
	}
}

// SetRPCClient sets the RPCClient for handling MCP request-response over gRPC stream.
func (c *GRPCConnection) SetRPCClient(rpc *RPCClient) {
	c.rpcClient = rpc
}

// GetRPCClient returns the RPCClient instance.
func (c *GRPCConnection) GetRPCClient() *RPCClient {
	return c.rpcClient
}
