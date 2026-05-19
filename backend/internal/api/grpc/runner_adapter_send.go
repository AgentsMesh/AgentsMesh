package grpc

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

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

func (a *GRPCRunnerAdapter) SendSubscribePod(runnerID int64, podKey, relayURL, runnerToken, localToken string, includeSnapshot bool, snapshotHistory int32) error {
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
				LocalToken:      localToken,
				IncludeSnapshot: includeSnapshot,
				SnapshotHistory: snapshotHistory,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return conn.SendMessage(msg)
}

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
