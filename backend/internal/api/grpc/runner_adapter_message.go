package grpc

import (
	"context"
	"time"

	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (a *GRPCRunnerAdapter) handleProtoMessage(ctx context.Context, runnerID int64, conn *runner.GRPCConnection, msg *runnerv1.RunnerMessage) {
	msgType := extractMessageType(msg)
	if !isHighFrequencyMessage(msgType) {
		otelinit.GRPCMessagesRecv.Add(ctx, 1, metric.WithAttributes(attribute.String("message.type", msgType)))
	}

	switch payload := msg.Payload.(type) {
	case *runnerv1.RunnerMessage_Initialize:
		a.handleInitialize(ctx, runnerID, conn, payload.Initialize)

	case *runnerv1.RunnerMessage_Initialized:
		a.handleInitialized(ctx, runnerID, conn, payload.Initialized)

	case *runnerv1.RunnerMessage_Heartbeat:
		a.connManager.HandleHeartbeat(runnerID, payload.Heartbeat)

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

		if len(payload.Heartbeat.AgentVersions) > 0 {
			a.handleHeartbeatAgentVersions(ctx, runnerID, payload.Heartbeat.AgentVersions)
		}

	case *runnerv1.RunnerMessage_PodCreated:
		a.connManager.HandlePodCreated(runnerID, payload.PodCreated)

	case *runnerv1.RunnerMessage_PodTerminated:
		a.connManager.HandlePodTerminated(runnerID, payload.PodTerminated)

	case *runnerv1.RunnerMessage_AgentStatus:
		a.connManager.HandleAgentStatus(runnerID, payload.AgentStatus)

	case *runnerv1.RunnerMessage_PodInitProgress:
		a.connManager.HandlePodInitProgress(runnerID, payload.PodInitProgress)

	case *runnerv1.RunnerMessage_Error:
		a.logger.Error("runner error",
			"runner_id", runnerID,
			"pod_key", payload.Error.PodKey,
			"code", payload.Error.Code,
			"message", payload.Error.Message,
		)
		a.connManager.HandlePodError(runnerID, payload.Error)

	case *runnerv1.RunnerMessage_RequestRelayToken:
		a.connManager.HandleRequestRelayToken(runnerID, payload.RequestRelayToken)

	case *runnerv1.RunnerMessage_SandboxesStatus:
		a.connManager.HandleSandboxesStatus(runnerID, payload.SandboxesStatus)

	case *runnerv1.RunnerMessage_OscNotification:
		a.connManager.HandleOSCNotification(runnerID, payload.OscNotification)

	case *runnerv1.RunnerMessage_OscTitle:
		a.connManager.HandleOSCTitle(runnerID, payload.OscTitle)

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
		a.connManager.HandleTokenUsage(runnerID, payload.TokenUsage)

	case *runnerv1.RunnerMessage_PodRestarting:
		a.connManager.HandlePodRestarting(runnerID, payload.PodRestarting)

	default:
		a.logger.Warn("unknown message type", "runner_id", runnerID)
	}
}
