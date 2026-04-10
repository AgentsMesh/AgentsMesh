package grpc

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

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

// ==================== Perpetual Pod Commands ====================

// SendUpdatePodPerpetual sends an update perpetual mode command to a Runner.
func (a *GRPCRunnerAdapter) SendUpdatePodPerpetual(runnerID int64, podKey string, perpetual bool) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UpdatePodPerpetual{
			UpdatePodPerpetual: &runnerv1.UpdatePodPerpetualCommand{
				PodKey:    podKey,
				Perpetual: perpetual,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}
