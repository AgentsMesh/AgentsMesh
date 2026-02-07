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

// SendTerminalInput sends terminal input to a pod.
func (a *GRPCRunnerAdapter) SendTerminalInput(runnerID int64, podKey string, data []byte) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalInput{
			TerminalInput: &runnerv1.TerminalInputCommand{
				PodKey: podKey,
				Data:   data,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminalResize sends terminal resize command to a pod.
func (a *GRPCRunnerAdapter) SendTerminalResize(runnerID int64, podKey string, cols, rows int32) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalResize{
			TerminalResize: &runnerv1.TerminalResizeCommand{
				PodKey: podKey,
				Cols:   cols,
				Rows:   rows,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

// SendTerminalRedraw sends terminal redraw command to a pod.
// This triggers SIGWINCH without changing terminal size, used for state recovery after server restart.
func (a *GRPCRunnerAdapter) SendTerminalRedraw(runnerID int64, podKey string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminalRedraw{
			TerminalRedraw: &runnerv1.TerminalRedrawCommand{
				PodKey: podKey,
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

// SendSubscribeTerminal sends a subscribe terminal command to a pod.
// This notifies the runner that a browser wants to observe the terminal via Relay.
func (a *GRPCRunnerAdapter) SendSubscribeTerminal(runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SubscribeTerminal{
			SubscribeTerminal: &runnerv1.SubscribeTerminalCommand{
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

// SendUnsubscribeTerminal sends an unsubscribe terminal command to a pod.
// This notifies the runner that all browsers have disconnected and it should disconnect from Relay.
func (a *GRPCRunnerAdapter) SendUnsubscribeTerminal(runnerID int64, podKey string) error {
	conn := a.connManager.GetConnection(runnerID)
	if conn == nil {
		return status.Errorf(codes.NotFound, "runner %d not connected", runnerID)
	}

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_UnsubscribeTerminal{
			UnsubscribeTerminal: &runnerv1.UnsubscribeTerminalCommand{
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
