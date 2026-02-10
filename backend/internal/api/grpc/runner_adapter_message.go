package grpc

import (
	"context"
	"time"

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
	}
}
