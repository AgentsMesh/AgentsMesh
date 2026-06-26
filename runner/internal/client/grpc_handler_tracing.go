package client

import (
	"context"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func extractServerMessageType(msg *runnerv1.ServerMessage) string {
	switch msg.Payload.(type) {
	case *runnerv1.ServerMessage_InitializeResult:
		return "InitializeResult"
	case *runnerv1.ServerMessage_CreatePod:
		return "CreatePod"
	case *runnerv1.ServerMessage_TerminatePod:
		return "TerminatePod"
	case *runnerv1.ServerMessage_SubscribePod:
		return "SubscribePod"
	case *runnerv1.ServerMessage_CreateAutopilot:
		return "CreateAutopilot"
	case *runnerv1.ServerMessage_PodInput:
		return "PodInput"
	case *runnerv1.ServerMessage_SendPrompt:
		return "SendPrompt"
	case *runnerv1.ServerMessage_UnsubscribePod:
		return "UnsubscribePod"
	case *runnerv1.ServerMessage_QuerySandboxes:
		return "QuerySandboxes"
	case *runnerv1.ServerMessage_ObservePod:
		return "ObservePod"
	case *runnerv1.ServerMessage_AutopilotControl:
		return "AutopilotControl"
	case *runnerv1.ServerMessage_McpResponse:
		return "McpResponse"
	case *runnerv1.ServerMessage_Ping:
		return "Ping"
	case *runnerv1.ServerMessage_HeartbeatAck:
		return "HeartbeatAck"
	case *runnerv1.ServerMessage_UpgradeRunner:
		return "UpgradeRunner"
	case *runnerv1.ServerMessage_UpgradeAgent:
		return "UpgradeAgent"
	case *runnerv1.ServerMessage_UploadLogs:
		return "UploadLogs"
	case *runnerv1.ServerMessage_UpdatePodPerpetual:
		return "UpdatePodPerpetual"
	default:
		return "Unknown"
	}
}

func isHighFrequencyServerMessage(msgType string) bool {
	switch msgType {
	case "HeartbeatAck", "Ping":
		return true
	default:
		return false
	}
}

func startMessageSpan(ctx context.Context, msgType string) (context.Context, trace.Span) {
	return otel.Tracer("agentsmesh-runner").Start(ctx, "grpc.recv."+msgType)
}
