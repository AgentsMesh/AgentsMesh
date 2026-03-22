package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// SendError sends an error event to the server (control message).
func (c *GRPCConnection) SendError(podKey, code, message string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendRequestRelayToken sends a request for a new relay token to the server.
// This is called when the relay connection fails due to token expiration.
func (c *GRPCConnection) SendRequestRelayToken(podKey, relayURL string) error {
	logger.GRPC().Info("Requesting new relay token", "pod_key", podKey)
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_RequestRelayToken{
			RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
				PodKey:   podKey,
				RelayUrl: relayURL,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendMessage sends a raw RunnerMessage to the server.
// Used for Autopilot events and other custom messages.
func (c *GRPCConnection) SendMessage(msg *runnerv1.RunnerMessage) error {
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	return c.sendControl(msg)
}

// SendSandboxesStatus sends sandbox status query response to the server (control message).
func (c *GRPCConnection) SendSandboxesStatus(requestID string, sandboxes []*SandboxStatusInfo) error {
	// Convert SandboxStatusInfo to proto
	protoSandboxes := make([]*runnerv1.SandboxStatus, len(sandboxes))
	for i, s := range sandboxes {
		protoSandboxes[i] = &runnerv1.SandboxStatus{
			PodKey:                s.PodKey,
			Exists:                s.Exists,
			SandboxPath:           s.SandboxPath,
			RepositoryUrl:         s.RepositoryURL,
			BranchName:            s.BranchName,
			CurrentCommit:         s.CurrentCommit,
			SizeBytes:             s.SizeBytes,
			LastModified:          s.LastModified,
			HasUncommittedChanges: s.HasUncommittedChanges,
			CanResume:             s.CanResume,
			Error:                 s.Error,
		}
	}

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_SandboxesStatus{
			SandboxesStatus: &runnerv1.SandboxesStatusEvent{
				RequestId: requestID,
				Sandboxes: protoSandboxes,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendObservePodResult sends terminal observation result to the server (control message).
func (c *GRPCConnection) SendObservePodResult(requestID, podKey, output, screen string, cursorX, cursorY, totalLines int, hasMore bool, errMsg string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_ObservePodResult{
			ObservePodResult: &runnerv1.ObservePodResult{
				RequestId:  requestID,
				PodKey:     podKey,
				Output:     output,
				Screen:     screen,
				CursorX:    int32(cursorX),
				CursorY:    int32(cursorY),
				TotalLines: int32(totalLines),
				HasMore:    hasMore,
				Error:      errMsg,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendUpgradeStatus sends an upgrade status event to the server (control message).
func (c *GRPCConnection) SendUpgradeStatus(event *runnerv1.UpgradeStatusEvent) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_UpgradeStatus{
			UpgradeStatus: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendLogUploadStatus sends a log upload status event to the server (control message).
func (c *GRPCConnection) SendLogUploadStatus(event *runnerv1.LogUploadStatusEvent) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_LogUploadStatus{
			LogUploadStatus: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// sendError sends an error event back to the server (internal use, control message).
func (c *GRPCConnection) sendError(podKey, code, message string) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.sendControl(msg); err != nil {
		logger.GRPC().Error("Failed to send error", "error", err)
	}
}
