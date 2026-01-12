package runner

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

// HandleMessage handles an incoming message from a runner
func (cm *ConnectionManager) HandleMessage(runnerID int64, msgType int, data []byte) {
	if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
		return
	}

	var msg RunnerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		cm.logger.Warn("failed to parse runner message",
			"runner_id", runnerID,
			"error", err)
		return
	}

	switch msg.Type {
	case MsgTypeHeartbeat:
		cm.handleHeartbeatMessage(runnerID, msg.Data)

	case MsgTypePodCreated:
		cm.handlePodCreatedMessage(runnerID, msg.Data)

	case MsgTypePodTerminated:
		cm.handlePodTerminatedMessage(runnerID, msg.Data)

	case MsgTypeTerminalOutput:
		cm.handleTerminalOutputMessage(runnerID, msg.Data)

	case MsgTypeAgentStatus:
		cm.handleAgentStatusMessage(runnerID, msg.Data)

	case MsgTypePtyResized:
		cm.handlePtyResizedMessage(runnerID, msg.Data)

	default:
		cm.logger.Debug("unknown message type",
			"runner_id", runnerID,
			"type", msg.Type)
	}
}

// handleHeartbeatMessage handles heartbeat message
func (cm *ConnectionManager) handleHeartbeatMessage(runnerID int64, data json.RawMessage) {
	var hbData HeartbeatData
	if err := json.Unmarshal(data, &hbData); err != nil {
		cm.logger.Error("failed to unmarshal heartbeat data",
			"runner_id", runnerID,
			"error", err,
			"data", string(data))
		return
	}

	cm.logger.Debug("received heartbeat",
		"runner_id", runnerID,
		"pods", len(hbData.Pods),
		"capabilities", len(hbData.Capabilities))
	cm.UpdateHeartbeat(runnerID)

	if cm.onHeartbeat != nil {
		cm.onHeartbeat(runnerID, &hbData)
	}
}

// handlePodCreatedMessage handles pod created message
func (cm *ConnectionManager) handlePodCreatedMessage(runnerID int64, data json.RawMessage) {
	var pcData PodCreatedData
	if err := json.Unmarshal(data, &pcData); err == nil {
		if cm.onPodCreated != nil {
			cm.onPodCreated(runnerID, &pcData)
		}
	}
}

// handlePodTerminatedMessage handles pod terminated message
func (cm *ConnectionManager) handlePodTerminatedMessage(runnerID int64, data json.RawMessage) {
	var ptData PodTerminatedData
	if err := json.Unmarshal(data, &ptData); err == nil {
		if cm.onPodTerminated != nil {
			cm.onPodTerminated(runnerID, &ptData)
		}
	}
}

// handleTerminalOutputMessage handles terminal output message
func (cm *ConnectionManager) handleTerminalOutputMessage(runnerID int64, data json.RawMessage) {
	var toData TerminalOutputData
	if err := json.Unmarshal(data, &toData); err == nil {
		if cm.onTerminalOutput != nil {
			cm.onTerminalOutput(runnerID, &toData)
		}
	}
}

// handleAgentStatusMessage handles agent status message
func (cm *ConnectionManager) handleAgentStatusMessage(runnerID int64, data json.RawMessage) {
	var asData AgentStatusData
	if err := json.Unmarshal(data, &asData); err == nil {
		if cm.onAgentStatus != nil {
			cm.onAgentStatus(runnerID, &asData)
		}
	}
}

// handlePtyResizedMessage handles PTY resized message
func (cm *ConnectionManager) handlePtyResizedMessage(runnerID int64, data json.RawMessage) {
	var prData PtyResizedData
	if err := json.Unmarshal(data, &prData); err == nil {
		if cm.onPtyResized != nil {
			cm.onPtyResized(runnerID, &prData)
		}
	}
}
