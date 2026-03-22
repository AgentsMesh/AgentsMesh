package grpc

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Send Operations (delegate to connection) ====================

// SendCreatePod sends a create pod command to a Runner.
func (a *GRPCRunnerAdapter) SendCreatePod(runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_CreatePod{
			CreatePod: cmd,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminatePod sends a terminate pod command to a Runner.
func (a *GRPCRunnerAdapter) SendTerminatePod(runnerID int64, podKey string, force bool) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminatePod{
			TerminatePod: &runnerv1.TerminatePodCommand{
				PodKey: podKey,
				Force:  force,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendPodInput sends pod input to a pod.
func (a *GRPCRunnerAdapter) SendPodInput(runnerID int64, podKey string, data []byte) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_PodInput{
			PodInput: &runnerv1.PodInputCommand{
				PodKey: podKey,
				Data:   data,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendPrompt sends a prompt to a pod.
func (a *GRPCRunnerAdapter) SendPrompt(runnerID int64, podKey, prompt string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SendPrompt{
			SendPrompt: &runnerv1.SendPromptCommand{
				PodKey: podKey,
				Prompt: prompt,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendSubscribePod sends a subscribe pod command to a pod.
// This notifies the runner that a browser wants to observe the pod via Relay.
func (a *GRPCRunnerAdapter) SendSubscribePod(runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SubscribePod{
			SubscribePod: &runnerv1.SubscribePodCommand{
				PodKey:          podKey,
				RelayUrl:        relayURL,
				RunnerToken:     runnerToken,
				IncludeSnapshot: includeSnapshot,
				SnapshotHistory: snapshotHistory,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendUnsubscribePod sends an unsubscribe pod command to a pod.
// This notifies the runner that all browsers have disconnected and it should disconnect from Relay.
func (a *GRPCRunnerAdapter) SendUnsubscribePod(runnerID int64, podKey string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UnsubscribePod{
			UnsubscribePod: &runnerv1.UnsubscribePodCommand{
				PodKey: podKey,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendQuerySandboxes sends a query sandboxes command to a Runner.
// Returns sandbox status for specified pod keys via callback registered in RunnerConnectionManager.
func (a *GRPCRunnerAdapter) SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	queries := make([]*runnerv1.SandboxQuery, len(podKeys))
	for i, podKey := range podKeys {
		queries[i] = &runnerv1.SandboxQuery{
			PodKey: podKey,
		}
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_QuerySandboxes{
			QuerySandboxes: &runnerv1.QuerySandboxesCommand{
				RequestId: requestID,
				Queries:   queries,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// ==================== Pod Observation Commands ====================

// SendObservePod sends an observe pod command to a Runner.
// Returns pod observation result via callback registered in RunnerConnectionManager.
func (a *GRPCRunnerAdapter) SendObservePod(runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_ObservePod{
			ObservePod: &runnerv1.ObservePodCommand{
				RequestId:     requestID,
				PodKey:        podKey,
				Lines:         lines,
				IncludeScreen: includeScreen,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// ==================== Runner Upgrade Commands ====================

// SendUpgradeRunner sends an upgrade command to a Runner.
func (a *GRPCRunnerAdapter) SendUpgradeRunner(runnerID int64, requestID, targetVersion string, force bool) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UpgradeRunner{
			UpgradeRunner: &runnerv1.UpgradeRunnerCommand{
				RequestId:     requestID,
				TargetVersion: targetVersion,
				Force:         force,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// ==================== Runner Log Upload Commands ====================

// SendUploadLogs sends a log upload command to a Runner.
func (a *GRPCRunnerAdapter) SendUploadLogs(runnerID int64, requestID, presignedURL string, urlExpiresAt int64) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UploadLogs{
			UploadLogs: &runnerv1.UploadLogsCommand{
				RequestId:    requestID,
				PresignedUrl: presignedURL,
				UrlExpiresAt: urlExpiresAt,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// ==================== AutopilotController Commands ====================

// SendCreateAutopilot sends a create AutopilotController command to a Runner.
func (a *GRPCRunnerAdapter) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_CreateAutopilot{
			CreateAutopilot: cmd,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendAutopilotControl sends an AutopilotController control command to a Runner.
func (a *GRPCRunnerAdapter) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_AutopilotControl{
			AutopilotControl: cmd,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}
