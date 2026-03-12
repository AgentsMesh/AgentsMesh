package grpc

import (
	"context"
	"time"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleProtoMessage routes proto messages directly to RunnerConnectionManager handlers.
// Zero-copy: Proto types are passed directly without JSON serialization.
func (a *GRPCRunnerAdapter) handleProtoMessage(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.RunnerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.RunnerMessage_Initialize:
		a.handleInitialize(ctx, runnerID, conn, payload.Initialize)

	case *runnerv1.RunnerMessage_Initialized:
		a.handleInitialized(ctx, runnerID, conn, payload.Initialized)

	case *runnerv1.RunnerMessage_Heartbeat:
		// Direct Proto type passing - no conversion
		a.connManager.HandleHeartbeat(runnerID, payload.Heartbeat)

		// Acknowledge heartbeat so Runner can detect upstream liveness.
		// Without this ack, Runner cannot distinguish "heartbeat arrived" from
		// "heartbeat was silently lost in a half-dead connection".
		ack := &runnerv1.ServerMessage{
			Payload: &runnerv1.ServerMessage_HeartbeatAck{
				HeartbeatAck: &runnerv1.HeartbeatAck{
					HeartbeatTimestamp: msg.Timestamp,
				},
			},
			Timestamp: time.Now().UnixMilli(),
		}
		if err := conn.SendMessage(ack); err != nil {
			a.logger.Warn("failed to send heartbeat ack", "runner_id", runnerID, "error", err)
		}

		// Process agent version updates from heartbeat (only present when versions changed)
		if len(payload.Heartbeat.AgentVersions) > 0 {
			a.handleHeartbeatAgentVersions(ctx, runnerID, payload.Heartbeat.AgentVersions)
		}

	case *runnerv1.RunnerMessage_PodCreated:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodCreated(runnerID, payload.PodCreated)

	case *runnerv1.RunnerMessage_PodTerminated:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodTerminated(runnerID, payload.PodTerminated)

	// NOTE: TerminalOutput case removed - terminal output is exclusively streamed via Relay.
	// Runner no longer sends TerminalOutputEvent via gRPC.

	case *runnerv1.RunnerMessage_AgentStatus:
		// Direct Proto type passing - no conversion
		a.connManager.HandleAgentStatus(runnerID, payload.AgentStatus)

	case *runnerv1.RunnerMessage_PtyResized:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePtyResized(runnerID, payload.PtyResized)

	case *runnerv1.RunnerMessage_PodInitProgress:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodInitProgress(runnerID, payload.PodInitProgress)

	case *runnerv1.RunnerMessage_Error:
		a.logger.Error("runner error",
			"runner_id", runnerID,
			"pod_key", payload.Error.PodKey,
			"code", payload.Error.Code,
			"message", payload.Error.Message,
		)
		// Route to callback chain for business processing (DB update, EventBus, WebSocket)
		a.connManager.HandlePodError(runnerID, payload.Error)

	case *runnerv1.RunnerMessage_RequestRelayToken:
		// Runner is requesting a new relay token (token expired during reconnection)
		a.connManager.HandleRequestRelayToken(runnerID, payload.RequestRelayToken)

	case *runnerv1.RunnerMessage_SandboxesStatus:
		// Direct Proto type passing - no conversion
		a.connManager.HandleSandboxesStatus(runnerID, payload.SandboxesStatus)

	case *runnerv1.RunnerMessage_OscNotification:
		// OSC 777/9 notification from terminal
		a.connManager.HandleOSCNotification(runnerID, payload.OscNotification)

	case *runnerv1.RunnerMessage_OscTitle:
		// OSC 0/2 title change from terminal
		a.connManager.HandleOSCTitle(runnerID, payload.OscTitle)

	// AutopilotController events
	case *runnerv1.RunnerMessage_AutopilotStatus:
		a.connManager.HandleAutopilotStatus(runnerID, payload.AutopilotStatus)

	case *runnerv1.RunnerMessage_AutopilotIteration:
		a.connManager.HandleAutopilotIteration(runnerID, payload.AutopilotIteration)

	case *runnerv1.RunnerMessage_AutopilotCreated:
		a.connManager.HandleAutopilotCreated(runnerID, payload.AutopilotCreated)

	case *runnerv1.RunnerMessage_AutopilotTerminated:
		a.connManager.HandleAutopilotTerminated(runnerID, payload.AutopilotTerminated)

	case *runnerv1.RunnerMessage_AutopilotThinking:
		a.connManager.HandleAutopilotThinking(runnerID, payload.AutopilotThinking)

	case *runnerv1.RunnerMessage_McpRequest:
		a.handleMcpRequest(ctx, runnerID, conn, payload.McpRequest)

	case *runnerv1.RunnerMessage_Pong:
		a.handlePong(runnerID, conn, payload.Pong)

	case *runnerv1.RunnerMessage_UpgradeStatus:
		a.connManager.HandleUpgradeStatus(runnerID, payload.UpgradeStatus)

	case *runnerv1.RunnerMessage_LogUploadStatus:
		a.connManager.HandleLogUploadStatus(runnerID, payload.LogUploadStatus)

	case *runnerv1.RunnerMessage_TokenUsage:
		// Token usage report from Runner (sent when pod exits)
		a.connManager.HandleTokenUsage(runnerID, payload.TokenUsage)

	default:
		a.logger.Warn("unknown message type", "runner_id", runnerID)
	}
}

// handleInitialize handles the initialize request - needs to send proto response
func (a *GRPCRunnerAdapter) handleInitialize(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, req *runnerv1.InitializeRequest) {
	a.logger.Debug("received initialize request",
		"runner_id", runnerID,
		"protocol_version", req.ProtocolVersion,
	)

	// Get agent types from provider
	var agentTypes []*runnerv1.AgentTypeInfo
	if a.agentTypesProvider != nil {
		types := a.agentTypesProvider.GetAgentTypesForRunner()
		agentTypes = make([]*runnerv1.AgentTypeInfo, len(types))
		for i, t := range types {
			agentTypes[i] = &runnerv1.AgentTypeInfo{
				Slug:    t.Slug,
				Name:    t.Name,
				Command: t.Executable,
			}
		}
		a.logger.Debug("sending agent types to runner",
			"runner_id", runnerID,
			"agent_types_count", len(agentTypes),
		)
	}

	// Persist runner version and host info from the handshake
	if req.RunnerInfo != nil && a.runnerService != nil {
		hostInfo := map[string]interface{}{
			"os":       req.RunnerInfo.GetOs(),
			"arch":     req.RunnerInfo.GetArch(),
			"hostname": req.RunnerInfo.GetHostname(),
		}
		if err := a.runnerService.UpdateRunnerVersionAndHostInfo(ctx, runnerID, req.RunnerInfo.GetVersion(), hostInfo); err != nil {
			a.logger.Error("Failed to update runner version and host info", "runner_id", runnerID, "error", err)
		}
	}

	// Build proto response
	result := &runnerv1.InitializeResult{
		ProtocolVersion: 2,
		ServerInfo: &runnerv1.ServerInfo{
			Version: "1.0.0",
		},
		AgentTypes: agentTypes,
		Features: []string{
			"files_to_create",
			"work_dir_config",
			"initial_prompt",
		},
	}

	response := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_InitializeResult{
			InitializeResult: result,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	// Send via connection's stream
	if err := conn.SendMessage(response); err != nil {
		a.logger.Warn("failed to send initialize result", "runner_id", runnerID, "error", err)
	}
}

// handleInitialized handles the initialized confirmation
func (a *GRPCRunnerAdapter) handleInitialized(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.InitializedConfirm) {
	a.logger.Info("Runner initialized",
		"runner_id", runnerID,
		"available_agents", msg.AvailableAgents,
		"agent_versions", len(msg.AgentVersions),
	)

	// Delegate to connManager for callback triggering (handles SetInitialized internally)
	a.connManager.HandleInitialized(runnerID, msg.AvailableAgents)

	// Update runner in database
	if a.runnerService != nil {
		_ = a.runnerService.UpdateLastSeen(ctx, runnerID)
		if err := a.runnerService.UpdateAvailableAgents(ctx, runnerID, msg.AvailableAgents); err != nil {
			a.logger.Error("failed to update available agents",
				"runner_id", runnerID,
				"error", err,
			)
		}

		// Save agent version info (backward compatible: old Runners won't send this)
		if len(msg.AgentVersions) > 0 {
			a.persistAgentVersions(ctx, runnerID, msg.AgentVersions, false)
		}
	}
}

// handleHeartbeatAgentVersions processes agent version changes reported in heartbeat.
// This is a delta update: only changed entries are included.
func (a *GRPCRunnerAdapter) handleHeartbeatAgentVersions(ctx context.Context, runnerID int64, versions []*runnerv1.AgentVersionInfo) {
	a.logger.Info("Agent version change detected via heartbeat",
		"runner_id", runnerID,
		"changes", len(versions),
	)
	a.persistAgentVersions(ctx, runnerID, versions, true)
}

// handlePong handles PongEvent from Runner - updates downstream liveness tracking.
func (a *GRPCRunnerAdapter) handlePong(runnerID int64, conn *runner.GRPCConnection, pong *runnerv1.PongEvent) {
	conn.UpdateLastPong()
	rtt := time.Now().UnixMilli() - pong.PingTimestamp
	a.logger.Debug("downstream pong received",
		"runner_id", runnerID,
		"rtt_ms", rtt,
	)
}

// persistAgentVersions saves agent version info to the database.
// If isDelta is true, merges with existing versions (heartbeat delta update).
// If isDelta is false, replaces all versions (initialization full report).
func (a *GRPCRunnerAdapter) persistAgentVersions(ctx context.Context, runnerID int64, versions []*runnerv1.AgentVersionInfo, isDelta bool) {
	if a.runnerService == nil {
		return
	}

	var finalVersions []runnerDomain.AgentVersion

	if isDelta {
		// Delta update: merge with existing versions in DB
		incoming := make(map[string]runnerDomain.AgentVersion, len(versions))
		for _, v := range versions {
			incoming[v.Slug] = runnerDomain.AgentVersion{
				Slug:    v.Slug,
				Version: v.Version,
				Path:    v.Path,
			}
			a.logger.Info("Agent version updated",
				"runner_id", runnerID,
				"agent", v.Slug,
				"version", v.Version,
				"path", v.Path,
			)
		}

		if err := a.runnerService.MergeAgentVersions(ctx, runnerID, incoming); err != nil {
			a.logger.Error("failed to merge agent versions",
				"runner_id", runnerID,
				"error", err,
			)
		}
		return
	}

	// Full update (initialization)
	finalVersions = make([]runnerDomain.AgentVersion, 0, len(versions))
	for _, v := range versions {
		finalVersions = append(finalVersions, runnerDomain.AgentVersion{
			Slug:    v.Slug,
			Version: v.Version,
			Path:    v.Path,
		})
		a.logger.Info("Agent version detected",
			"runner_id", runnerID,
			"agent", v.Slug,
			"version", v.Version,
			"path", v.Path,
		)
	}
	if err := a.runnerService.UpdateAgentVersions(ctx, runnerID, finalVersions); err != nil {
		a.logger.Error("failed to update agent versions",
			"runner_id", runnerID,
			"error", err,
		)
	}
}
