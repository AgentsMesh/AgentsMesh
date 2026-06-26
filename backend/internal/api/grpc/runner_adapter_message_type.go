package grpc

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func extractMessageType(msg *runnerv1.RunnerMessage) string {
	switch msg.Payload.(type) {
	case *runnerv1.RunnerMessage_Initialize:
		return "Initialize"
	case *runnerv1.RunnerMessage_Initialized:
		return "Initialized"
	case *runnerv1.RunnerMessage_Heartbeat:
		return "Heartbeat"
	case *runnerv1.RunnerMessage_PodCreated:
		return "PodCreated"
	case *runnerv1.RunnerMessage_PodTerminated:
		return "PodTerminated"
	case *runnerv1.RunnerMessage_AgentStatus:
		return "AgentStatus"
	case *runnerv1.RunnerMessage_PodInitProgress:
		return "PodInitProgress"
	case *runnerv1.RunnerMessage_Error:
		return "Error"
	case *runnerv1.RunnerMessage_RequestRelayToken:
		return "RequestRelayToken"
	case *runnerv1.RunnerMessage_SandboxesStatus:
		return "SandboxesStatus"
	case *runnerv1.RunnerMessage_OscNotification:
		return "OscNotification"
	case *runnerv1.RunnerMessage_OscTitle:
		return "OscTitle"
	case *runnerv1.RunnerMessage_AutopilotStatus:
		return "AutopilotStatus"
	case *runnerv1.RunnerMessage_AutopilotIteration:
		return "AutopilotIteration"
	case *runnerv1.RunnerMessage_AutopilotCreated:
		return "AutopilotCreated"
	case *runnerv1.RunnerMessage_AutopilotTerminated:
		return "AutopilotTerminated"
	case *runnerv1.RunnerMessage_AutopilotThinking:
		return "AutopilotThinking"
	case *runnerv1.RunnerMessage_ObservePodResult:
		return "ObservePodResult"
	case *runnerv1.RunnerMessage_McpRequest:
		return "McpRequest"
	case *runnerv1.RunnerMessage_Pong:
		return "Pong"
	case *runnerv1.RunnerMessage_UpgradeStatus:
		return "UpgradeStatus"
	case *runnerv1.RunnerMessage_UpgradeAgentResult:
		return "UpgradeAgentResult"
	case *runnerv1.RunnerMessage_LogUploadStatus:
		return "LogUploadStatus"
	case *runnerv1.RunnerMessage_TokenUsage:
		return "TokenUsage"
	case *runnerv1.RunnerMessage_PodRestarting:
		return "PodRestarting"
	default:
		return "Unknown"
	}
}

func isHighFrequencyMessage(msgType string) bool {
	switch msgType {
	case "Heartbeat", "Pong":
		return true
	default:
		return false
	}
}
