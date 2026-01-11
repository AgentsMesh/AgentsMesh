package runner

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionManager manages runner WebSocket connections
type ConnectionManager struct {
	connections  map[int64]*RunnerConnection
	mu           sync.RWMutex
	logger       *slog.Logger
	pingInterval time.Duration
	pingTimeout  time.Duration

	// Event callbacks
	onHeartbeat      func(runnerID int64, data *HeartbeatData)
	onPodCreated     func(runnerID int64, data *PodCreatedData)
	onPodTerminated  func(runnerID int64, data *PodTerminatedData)
	onTerminalOutput func(runnerID int64, data *TerminalOutputData)
	onAgentStatus    func(runnerID int64, data *AgentStatusData)
	onPtyResized     func(runnerID int64, data *PtyResizedData)
	onDisconnect     func(runnerID int64)
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(logger *slog.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections:  make(map[int64]*RunnerConnection),
		logger:       logger,
		pingInterval: 30 * time.Second,
		pingTimeout:  60 * time.Second,
	}
}

// ========== Callback Setters ==========

// SetHeartbeatCallback sets the heartbeat callback
func (cm *ConnectionManager) SetHeartbeatCallback(fn func(runnerID int64, data *HeartbeatData)) {
	cm.onHeartbeat = fn
}

// SetPodCreatedCallback sets the pod created callback
func (cm *ConnectionManager) SetPodCreatedCallback(fn func(runnerID int64, data *PodCreatedData)) {
	cm.onPodCreated = fn
}

// SetPodTerminatedCallback sets the pod terminated callback
func (cm *ConnectionManager) SetPodTerminatedCallback(fn func(runnerID int64, data *PodTerminatedData)) {
	cm.onPodTerminated = fn
}

// SetTerminalOutputCallback sets the terminal output callback
func (cm *ConnectionManager) SetTerminalOutputCallback(fn func(runnerID int64, data *TerminalOutputData)) {
	cm.onTerminalOutput = fn
}

// SetAgentStatusCallback sets the agent status callback
func (cm *ConnectionManager) SetAgentStatusCallback(fn func(runnerID int64, data *AgentStatusData)) {
	cm.onAgentStatus = fn
}

// SetPtyResizedCallback sets the PTY resized callback
func (cm *ConnectionManager) SetPtyResizedCallback(fn func(runnerID int64, data *PtyResizedData)) {
	cm.onPtyResized = fn
}

// SetDisconnectCallback sets the disconnect callback
func (cm *ConnectionManager) SetDisconnectCallback(fn func(runnerID int64)) {
	cm.onDisconnect = fn
}

// GetHeartbeatCallback returns the current heartbeat callback
func (cm *ConnectionManager) GetHeartbeatCallback() func(runnerID int64, data *HeartbeatData) {
	return cm.onHeartbeat
}

// GetDisconnectCallback returns the current disconnect callback
func (cm *ConnectionManager) GetDisconnectCallback() func(runnerID int64) {
	return cm.onDisconnect
}

// ========== Connection Management ==========

// AddConnection adds a runner connection
func (cm *ConnectionManager) AddConnection(runnerID int64, conn *websocket.Conn) *RunnerConnection {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Close existing connection if any
	if existing, ok := cm.connections[runnerID]; ok {
		existing.Close()
	}

	rc := &RunnerConnection{
		RunnerID: runnerID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
	}

	cm.connections[runnerID] = rc
	cm.logger.Info("runner connected", "runner_id", runnerID)

	return rc
}

// RemoveConnection removes a runner connection
func (cm *ConnectionManager) RemoveConnection(runnerID int64) {
	cm.mu.Lock()
	conn, ok := cm.connections[runnerID]
	if ok {
		delete(cm.connections, runnerID)
	}
	cm.mu.Unlock()

	if ok {
		conn.Close()
		cm.logger.Info("runner disconnected", "runner_id", runnerID)

		if cm.onDisconnect != nil {
			cm.onDisconnect(runnerID)
		}
	}
}

// GetConnection returns a runner connection
func (cm *ConnectionManager) GetConnection(runnerID int64) *RunnerConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connections[runnerID]
}

// IsConnected checks if a runner is connected
func (cm *ConnectionManager) IsConnected(runnerID int64) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.connections[runnerID]
	return ok
}

// UpdateHeartbeat updates the last ping time for a runner
func (cm *ConnectionManager) UpdateHeartbeat(runnerID int64) {
	cm.mu.RLock()
	conn, ok := cm.connections[runnerID]
	cm.mu.RUnlock()

	if ok {
		conn.mu.Lock()
		conn.LastPing = time.Now()
		conn.mu.Unlock()
	}
}

// GetConnectedRunnerIDs returns IDs of all connected runners
func (cm *ConnectionManager) GetConnectedRunnerIDs() []int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]int64, 0, len(cm.connections))
	for id := range cm.connections {
		ids = append(ids, id)
	}
	return ids
}

// Close closes the connection manager and all connections
func (cm *ConnectionManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, conn := range cm.connections {
		conn.Close()
	}
	cm.connections = make(map[int64]*RunnerConnection)
}

// ========== Message Handling ==========

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
		var hbData HeartbeatData
		if err := json.Unmarshal(msg.Data, &hbData); err == nil {
			cm.UpdateHeartbeat(runnerID)
			if cm.onHeartbeat != nil {
				cm.onHeartbeat(runnerID, &hbData)
			}
		}

	case MsgTypePodCreated:
		var pcData PodCreatedData
		if err := json.Unmarshal(msg.Data, &pcData); err == nil {
			if cm.onPodCreated != nil {
				cm.onPodCreated(runnerID, &pcData)
			}
		}

	case MsgTypePodTerminated:
		var ptData PodTerminatedData
		if err := json.Unmarshal(msg.Data, &ptData); err == nil {
			if cm.onPodTerminated != nil {
				cm.onPodTerminated(runnerID, &ptData)
			}
		}

	case MsgTypeTerminalOutput:
		var toData TerminalOutputData
		if err := json.Unmarshal(msg.Data, &toData); err == nil {
			if cm.onTerminalOutput != nil {
				cm.onTerminalOutput(runnerID, &toData)
			}
		}

	case MsgTypeAgentStatus:
		var asData AgentStatusData
		if err := json.Unmarshal(msg.Data, &asData); err == nil {
			if cm.onAgentStatus != nil {
				cm.onAgentStatus(runnerID, &asData)
			}
		}

	case MsgTypePtyResized:
		var prData PtyResizedData
		if err := json.Unmarshal(msg.Data, &prData); err == nil {
			if cm.onPtyResized != nil {
				cm.onPtyResized(runnerID, &prData)
			}
		}

	default:
		cm.logger.Debug("unknown message type",
			"runner_id", runnerID,
			"type", msg.Type)
	}
}

// ========== Send Operations ==========

// SendMessage sends a message to a runner
func (cm *ConnectionManager) SendMessage(ctx context.Context, runnerID int64, msg *RunnerMessage) error {
	cm.mu.RLock()
	conn, ok := cm.connections[runnerID]
	cm.mu.RUnlock()

	if !ok {
		return ErrRunnerNotConnected
	}

	return conn.SendMessage(msg)
}

// SendCreatePod sends a create pod request to a runner
func (cm *ConnectionManager) SendCreatePod(ctx context.Context, runnerID int64, req *CreatePodRequest) error {
	cm.logger.Info("sending create_pod to runner",
		"runner_id", runnerID,
		"pod_key", req.PodKey,
		"initial_command", req.InitialCommand,
		"permission_mode", req.PermissionMode)

	data, err := json.Marshal(req)
	if err != nil {
		cm.logger.Error("failed to marshal create_pod request", "error", err)
		return err
	}

	err = cm.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeCreatePod,
		PodKey:    req.PodKey,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	})

	if err != nil {
		cm.logger.Error("failed to send create_pod to runner",
			"runner_id", runnerID,
			"pod_key", req.PodKey,
			"error", err)
	} else {
		cm.logger.Info("create_pod sent successfully",
			"runner_id", runnerID,
			"pod_key", req.PodKey)
	}

	return err
}

// SendTerminatePod sends a terminate pod request to a runner
func (cm *ConnectionManager) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	return cm.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminatePod,
		PodKey:    podKey,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendTerminalInput sends terminal input to a runner
func (cm *ConnectionManager) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	inputData, _ := json.Marshal(&TerminalInputRequest{
		PodKey: podKey,
		Data:   data,
	})

	return cm.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminalInput,
		PodKey:    podKey,
		Data:      inputData,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendTerminalResize sends terminal resize to a runner
func (cm *ConnectionManager) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	resizeData, _ := json.Marshal(&TerminalResizeRequest{
		PodKey: podKey,
		Cols:   cols,
		Rows:   rows,
	})

	return cm.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeTerminalResize,
		PodKey:    podKey,
		Data:      resizeData,
		Timestamp: time.Now().UnixMilli(),
	})
}

// SendPrompt sends a prompt to a pod
func (cm *ConnectionManager) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	promptData, _ := json.Marshal(map[string]string{
		"pod_key": podKey,
		"prompt":  prompt,
	})

	return cm.SendMessage(ctx, runnerID, &RunnerMessage{
		Type:      MsgTypeSendPrompt,
		PodKey:    podKey,
		Data:      promptData,
		Timestamp: time.Now().UnixMilli(),
	})
}
