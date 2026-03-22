// Package client provides gRPC connection management for Runner.
package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// handlePodInput handles pod_input command from server.
func (c *GRPCConnection) handlePodInput(cmd *runnerv1.PodInputCommand) {
	if c.handler == nil {
		return
	}

	req := PodInputRequest{
		PodKey: cmd.PodKey,
		Data:   cmd.Data, // gRPC uses native bytes, no encoding needed
	}
	if err := c.handler.OnPodInput(req); err != nil {
		logger.GRPC().Error("Failed to send input to pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSendPrompt handles send_prompt command from server.
func (c *GRPCConnection) handleSendPrompt(cmd *runnerv1.SendPromptCommand) {
	log := logger.GRPC()
	log.Debug("Received send_prompt", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring send_prompt")
		return
	}
	if err := c.handler.OnSendPrompt(cmd); err != nil {
		log.Error("Failed to handle send_prompt", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSubscribePod handles subscribe_pod command from server.
// This notifies the Runner that a browser wants to observe the pod via Relay.
// Channel is identified by PodKey (not session ID).
func (c *GRPCConnection) handleSubscribePod(cmd *runnerv1.SubscribePodCommand) {
	log := logger.GRPC()
	log.Info("Received subscribe_pod", "pod_key", cmd.PodKey, "relay_url", cmd.RelayUrl)
	if c.handler == nil {
		log.Warn("No handler set, ignoring subscribe_pod")
		return
	}

	req := SubscribePodRequest{
		PodKey:          cmd.PodKey,
		RelayURL:        cmd.RelayUrl,
		RunnerToken:     cmd.RunnerToken,
		IncludeSnapshot: cmd.IncludeSnapshot,
		SnapshotHistory: cmd.SnapshotHistory,
	}
	if err := c.handler.OnSubscribePod(req); err != nil {
		log.Error("Failed to subscribe pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleUnsubscribePod handles unsubscribe_pod command from server.
// This notifies the Runner that all browsers have disconnected.
func (c *GRPCConnection) handleUnsubscribePod(cmd *runnerv1.UnsubscribePodCommand) {
	log := logger.GRPC()
	log.Info("Received unsubscribe_pod", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring unsubscribe_pod")
		return
	}

	req := UnsubscribePodRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnUnsubscribePod(req); err != nil {
		log.Error("Failed to unsubscribe pod", "pod_key", cmd.PodKey, "error", err)
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

// handleObservePod handles observe_pod (get_pod_snapshot) command from server.
// Reads VirtualTerminal state and sends result back via gRPC.
func (c *GRPCConnection) handleObservePod(cmd *runnerv1.ObservePodCommand) {
	log := logger.GRPC()
	log.Info("Received get_pod_snapshot request", "request_id", cmd.RequestId, "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring get_pod_snapshot request")
		return
	}

	req := ObservePodRequest{
		RequestID:     cmd.RequestId,
		PodKey:        cmd.PodKey,
		Lines:         int(cmd.Lines),
		IncludeScreen: cmd.IncludeScreen,
	}
	if err := c.handler.OnObservePod(req); err != nil {
		log.Error("Failed to observe pod", "request_id", cmd.RequestId, "pod_key", cmd.PodKey, "error", err)
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

// handleUploadLogs handles upload_logs command from server.
func (c *GRPCConnection) handleUploadLogs(cmd *runnerv1.UploadLogsCommand) {
	log := logger.GRPC()
	log.Info("Received upload_logs", "request_id", cmd.RequestId)
	if c.handler == nil {
		log.Warn("No handler set, ignoring upload_logs")
		return
	}

	if err := c.handler.OnUploadLogs(cmd); err != nil {
		log.Error("Failed to handle upload logs", "request_id", cmd.RequestId, "error", err)
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
