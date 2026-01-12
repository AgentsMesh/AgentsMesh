package runner

import (
	"context"
	"encoding/json"
	"time"
)

// RunnerSender provides methods to send messages to runners
// It wraps ConnectionManager to provide a cleaner API for message sending operations
type RunnerSender struct {
	cm *ConnectionManager
}

// NewRunnerSender creates a new RunnerSender
func NewRunnerSender(cm *ConnectionManager) *RunnerSender {
	return &RunnerSender{cm: cm}
}

// SendMessage sends a message to a runner
func (rs *RunnerSender) SendMessage(ctx context.Context, runnerID int64, msg *RunnerMessage) error {
	shard := rs.cm.getShard(runnerID)

	shard.mu.RLock()
	conn, ok := shard.connections[runnerID]
	shard.mu.RUnlock()

	if !ok {
		return ErrRunnerNotConnected
	}

	return conn.SendMessage(msg)
}

// SendCreatePod sends a create pod request to a runner
func (rs *RunnerSender) SendCreatePod(ctx context.Context, runnerID int64, req *CreatePodRequest) error {
	rs.cm.logger.Info("sending create_pod to runner",
		"runner_id", runnerID,
		"pod_key", req.PodKey,
		"initial_command", req.InitialCommand,
		"permission_mode", req.PermissionMode)

	data, err := json.Marshal(req)
	if err != nil {
		rs.cm.logger.Error("failed to marshal create_pod request", "error", err)
		return err
	}

	err = rs.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeCreatePod,
		PodKey:    req.PodKey,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})

	if err != nil {
		rs.cm.logger.Error("failed to send create_pod to runner",
			"runner_id", runnerID,
			"pod_key", req.PodKey,
			"error", err)
	} else {
		rs.cm.logger.Info("create_pod sent successfully",
			"runner_id", runnerID,
			"pod_key", req.PodKey)
	}

	return err
}

// SendTerminatePod sends a terminate pod request to a runner
func (rs *RunnerSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	return rs.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminatePod,
		PodKey:    podKey,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendTerminalInput sends terminal input to a runner
func (rs *RunnerSender) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	inputData, _ := json.Marshal(&TerminalInputRequest{
		PodKey: podKey,
		Data:   data,
	})

	return rs.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminalInput,
		PodKey:    podKey,
		Data:      inputData,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendTerminalResize sends terminal resize to a runner
func (rs *RunnerSender) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	resizeData, _ := json.Marshal(&TerminalResizeRequest{
		PodKey: podKey,
		Cols:   cols,
		Rows:   rows,
	})

	return rs.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminalResize,
		PodKey:    podKey,
		Data:      resizeData,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendPrompt sends a prompt to a pod
func (rs *RunnerSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	promptData, _ := json.Marshal(map[string]string{
		"pod_key": podKey,
		"prompt":  prompt,
	})

	return rs.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeSendPrompt,
		PodKey:    podKey,
		Data:      promptData,
		Timestamp: time.Now().UnixMilli(),
	})
}
