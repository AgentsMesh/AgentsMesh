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

	// NOTE: PodOutput case removed - output is exclusively streamed via Relay.
	// Runner no longer sends PodOutputEvent via gRPC.

	case *runnerv1.RunnerMessage_AgentStatus:
		// Direct Proto type passing - no conversion
		a.connManager.HandleAgentStatus(runnerID, payload.AgentStatus)

	case *runnerv1.RunnerMessage_PodResized:
		// Direct Proto type passing - no conversion
		a.connManager.HandlePodResized(runnerID, payload.PodResized)

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

	case *runnerv1.RunnerMessage_ObservePodResult:
		// Direct Proto type passing - no conversion
		a.connManager.HandleObservePodResult(runnerID, payload.ObservePodResult)

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

// Handler implementations (handleInitialize, handleInitialized, handlePong, etc.)
// are in runner_adapter_message_helpers.go
