package runner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewConnectionManager(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	if cm == nil {
		t.Fatal("NewConnectionManager returned nil")
	}
	if cm.shards[0] == nil {
		t.Error("shards should be initialized")
	}
	if cm.pingInterval != 30*time.Second {
		t.Errorf("pingInterval = %v, want 30s", cm.pingInterval)
	}
	if cm.pingTimeout != 60*time.Second {
		t.Errorf("pingTimeout = %v, want 60s", cm.pingTimeout)
	}
	if cm.ConnectionCount() != 0 {
		t.Errorf("initial connection count should be 0, got %d", cm.ConnectionCount())
	}
}

func TestConnectionManagerSetCallbacks(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	heartbeatCalled := false
	cm.SetHeartbeatCallback(func(runnerID int64, data *HeartbeatData) {
		heartbeatCalled = true
	})
	if cm.onHeartbeat == nil {
		t.Error("onHeartbeat should be set")
	}

	podCreatedCalled := false
	cm.SetPodCreatedCallback(func(runnerID int64, data *PodCreatedData) {
		podCreatedCalled = true
	})
	if cm.onPodCreated == nil {
		t.Error("onPodCreated should be set")
	}

	podTerminatedCalled := false
	cm.SetPodTerminatedCallback(func(runnerID int64, data *PodTerminatedData) {
		podTerminatedCalled = true
	})
	if cm.onPodTerminated == nil {
		t.Error("onPodTerminated should be set")
	}

	terminalOutputCalled := false
	cm.SetTerminalOutputCallback(func(runnerID int64, data *TerminalOutputData) {
		terminalOutputCalled = true
	})
	if cm.onTerminalOutput == nil {
		t.Error("onTerminalOutput should be set")
	}

	agentStatusCalled := false
	cm.SetAgentStatusCallback(func(runnerID int64, data *AgentStatusData) {
		agentStatusCalled = true
	})
	if cm.onAgentStatus == nil {
		t.Error("onAgentStatus should be set")
	}

	ptyResizedCalled := false
	cm.SetPtyResizedCallback(func(runnerID int64, data *PtyResizedData) {
		ptyResizedCalled = true
	})
	if cm.onPtyResized == nil {
		t.Error("onPtyResized should be set")
	}

	disconnectCalled := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnectCalled = true
	})
	if cm.onDisconnect == nil {
		t.Error("onDisconnect should be set")
	}

	// Test they are not called yet
	if heartbeatCalled || podCreatedCalled || podTerminatedCalled ||
		terminalOutputCalled || agentStatusCalled || ptyResizedCalled || disconnectCalled {
		t.Error("callbacks should not be called yet")
	}
}

func TestConnectionManagerIsConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	if cm.IsConnected(1) {
		t.Error("runner 1 should not be connected")
	}

	// Add a mock connection using shard directly for testing
	shard := cm.getShard(1)
	shard.mu.Lock()
	shard.connections[1] = &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard.mu.Unlock()

	if !cm.IsConnected(1) {
		t.Error("runner 1 should be connected")
	}
}

func TestConnectionManagerGetConnection(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	conn := cm.GetConnection(1)
	if conn != nil {
		t.Error("should return nil for nonexistent connection")
	}

	mockConn := &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	shard := cm.getShard(1)
	shard.mu.Lock()
	shard.connections[1] = mockConn
	cm.connCount.Add(1)
	shard.mu.Unlock()

	conn = cm.GetConnection(1)
	if conn != mockConn {
		t.Error("should return the connection")
	}
}

func TestConnectionManagerUpdateHeartbeat(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	// Update nonexistent - should not panic
	cm.UpdateHeartbeat(1)

	// Add connection and update
	mockConn := &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	shard := cm.getShard(1)
	shard.mu.Lock()
	shard.connections[1] = mockConn
	cm.connCount.Add(1)
	shard.mu.Unlock()

	before := time.Now()
	cm.UpdateHeartbeat(1)
	after := time.Now()

	if mockConn.LastPing.Before(before) || mockConn.LastPing.After(after) {
		t.Error("LastPing should be updated to current time")
	}
}

func TestConnectionManagerGetConnectedRunnerIDs(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	ids := cm.GetConnectedRunnerIDs()
	if len(ids) != 0 {
		t.Errorf("expected 0 runners, got %d", len(ids))
	}

	// Add connections to different shards
	shard1 := cm.getShard(1)
	shard1.mu.Lock()
	shard1.connections[1] = &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard1.mu.Unlock()

	shard2 := cm.getShard(2)
	shard2.mu.Lock()
	shard2.connections[2] = &RunnerConnection{RunnerID: 2, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard2.mu.Unlock()

	ids = cm.GetConnectedRunnerIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 runners, got %d", len(ids))
	}
}

func TestConnectionManagerClose(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	shard1 := cm.getShard(1)
	shard1.mu.Lock()
	shard1.connections[1] = &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard1.mu.Unlock()

	shard2 := cm.getShard(2)
	shard2.mu.Lock()
	shard2.connections[2] = &RunnerConnection{RunnerID: 2, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard2.mu.Unlock()

	cm.Close()

	if cm.ConnectionCount() != 0 {
		t.Errorf("connections should be empty after Close, got %d", cm.ConnectionCount())
	}
}

func TestConnectionManagerRemoveConnection(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	disconnectCalled := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnectCalled = true
	})

	// Remove nonexistent - should not panic
	cm.RemoveConnection(1)
	if disconnectCalled {
		t.Error("disconnect callback should not be called for nonexistent connection")
	}

	// Add and remove
	shard := cm.getShard(1)
	shard.mu.Lock()
	shard.connections[1] = &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard.mu.Unlock()

	cm.RemoveConnection(1)
	if !disconnectCalled {
		t.Error("disconnect callback should be called")
	}
	if cm.IsConnected(1) {
		t.Error("connection should be removed")
	}
}

func TestConnectionManagerHandleMessageInvalidType(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	// PingMessage should be ignored
	cm.HandleMessage(1, websocket.PingMessage, []byte{})
}

func TestConnectionManagerHandleMessageInvalidJSON(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())
	// Should not panic
	cm.HandleMessage(1, websocket.TextMessage, []byte("invalid json"))
}

func TestConnectionManagerHandleMessageHeartbeat(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	heartbeatCalled := false
	var receivedData *HeartbeatData
	cm.SetHeartbeatCallback(func(runnerID int64, data *HeartbeatData) {
		heartbeatCalled = true
		receivedData = data
	})

	// Add mock connection for heartbeat update
	shard := cm.getShard(1)
	shard.mu.Lock()
	shard.connections[1] = &RunnerConnection{RunnerID: 1, Send: make(chan []byte, 256)}
	cm.connCount.Add(1)
	shard.mu.Unlock()

	hbData := HeartbeatData{
		RunnerVersion: "1.0.0",
		Pods:          []HeartbeatPod{{PodKey: "s1"}},
	}
	dataBytes, _ := json.Marshal(hbData)
	msg := RunnerMessage{
		Type: MsgTypeHeartbeat,
		Data: dataBytes,
	}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !heartbeatCalled {
		t.Error("heartbeat callback should be called")
	}
	if receivedData == nil || receivedData.RunnerVersion != "1.0.0" {
		t.Error("heartbeat data not received correctly")
	}
}

func TestConnectionManagerHandleMessagePodCreated(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	podCreatedCalled := false
	cm.SetPodCreatedCallback(func(runnerID int64, data *PodCreatedData) {
		podCreatedCalled = true
	})

	scData := PodCreatedData{PodKey: "s1", Pid: 12345}
	dataBytes, _ := json.Marshal(scData)
	msg := RunnerMessage{Type: MsgTypePodCreated, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !podCreatedCalled {
		t.Error("pod created callback should be called")
	}
}

func TestConnectionManagerHandleMessagePodTerminated(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	podTerminatedCalled := false
	cm.SetPodTerminatedCallback(func(runnerID int64, data *PodTerminatedData) {
		podTerminatedCalled = true
	})

	stData := PodTerminatedData{PodKey: "s1", ExitCode: 0}
	dataBytes, _ := json.Marshal(stData)
	msg := RunnerMessage{Type: MsgTypePodTerminated, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !podTerminatedCalled {
		t.Error("pod terminated callback should be called")
	}
}

func TestConnectionManagerHandleMessageTerminalOutput(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	terminalOutputCalled := false
	cm.SetTerminalOutputCallback(func(runnerID int64, data *TerminalOutputData) {
		terminalOutputCalled = true
	})

	toData := TerminalOutputData{PodKey: "s1", Data: []byte("output")}
	dataBytes, _ := json.Marshal(toData)
	msg := RunnerMessage{Type: MsgTypeTerminalOutput, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !terminalOutputCalled {
		t.Error("terminal output callback should be called")
	}
}

func TestConnectionManagerHandleMessageAgentStatus(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	agentStatusCalled := false
	cm.SetAgentStatusCallback(func(runnerID int64, data *AgentStatusData) {
		agentStatusCalled = true
	})

	asData := AgentStatusData{PodKey: "s1", Status: "running"}
	dataBytes, _ := json.Marshal(asData)
	msg := RunnerMessage{Type: MsgTypeAgentStatus, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !agentStatusCalled {
		t.Error("agent status callback should be called")
	}
}

func TestConnectionManagerHandleMessagePtyResized(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	ptyResizedCalled := false
	cm.SetPtyResizedCallback(func(runnerID int64, data *PtyResizedData) {
		ptyResizedCalled = true
	})

	prData := PtyResizedData{PodKey: "s1", Cols: 80, Rows: 24}
	dataBytes, _ := json.Marshal(prData)
	msg := RunnerMessage{Type: MsgTypePtyResized, Data: dataBytes}
	msgBytes, _ := json.Marshal(msg)

	cm.HandleMessage(1, websocket.TextMessage, msgBytes)

	if !ptyResizedCalled {
		t.Error("PTY resized callback should be called")
	}
}

func TestConnectionManagerHandleMessageUnknown(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	msg := RunnerMessage{Type: "unknown_type"}
	msgBytes, _ := json.Marshal(msg)

	// Should not panic
	cm.HandleMessage(1, websocket.TextMessage, msgBytes)
}

func TestConnectionManagerSendMessageNotConnected(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	err := cm.SendMessage(nil, 1, &RunnerMessage{Type: MsgTypeCreatePod})
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

func TestRunnerConnectionSendMessage(t *testing.T) {
	rc := &RunnerConnection{
		RunnerID: 1,
		Send:     make(chan []byte, 256),
	}

	// Connection nil
	err := rc.SendMessage(&RunnerMessage{Type: MsgTypeCreatePod})
	if err != ErrConnectionClosed {
		t.Errorf("err = %v, want ErrConnectionClosed", err)
	}
}

func TestRunnerConnectionClose(t *testing.T) {
	rc := &RunnerConnection{
		RunnerID: 1,
		Conn:     nil,
		Send:     make(chan []byte, 256),
	}

	// Should not panic
	rc.Close()
}

func TestRunnerMessageStruct(t *testing.T) {
	msg := RunnerMessage{
		Type:      MsgTypeHeartbeat,
		PodKey:    "pod-1",
		Timestamp: time.Now().UnixMilli(),
	}

	if msg.Type != MsgTypeHeartbeat {
		t.Errorf("Type = %s, want %s", msg.Type, MsgTypeHeartbeat)
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// From runner
	if MsgTypeHeartbeat != "heartbeat" {
		t.Errorf("MsgTypeHeartbeat = %s, want heartbeat", MsgTypeHeartbeat)
	}
	if MsgTypePodCreated != "pod_created" {
		t.Errorf("MsgTypePodCreated = %s, want pod_created", MsgTypePodCreated)
	}
	if MsgTypePodTerminated != "pod_terminated" {
		t.Errorf("MsgTypePodTerminated = %s, want pod_terminated", MsgTypePodTerminated)
	}
	if MsgTypeTerminalOutput != "terminal_output" {
		t.Errorf("MsgTypeTerminalOutput = %s, want terminal_output", MsgTypeTerminalOutput)
	}
	if MsgTypeAgentStatus != "agent_status" {
		t.Errorf("MsgTypeAgentStatus = %s, want agent_status", MsgTypeAgentStatus)
	}
	if MsgTypePtyResized != "pty_resized" {
		t.Errorf("MsgTypePtyResized = %s, want pty_resized", MsgTypePtyResized)
	}
	if MsgTypeError != "error" {
		t.Errorf("MsgTypeError = %s, want error", MsgTypeError)
	}

	// To runner
	if MsgTypeCreatePod != "create_pod" {
		t.Errorf("MsgTypeCreatePod = %s, want create_pod", MsgTypeCreatePod)
	}
	if MsgTypeTerminatePod != "terminate_pod" {
		t.Errorf("MsgTypeTerminatePod = %s, want terminate_pod", MsgTypeTerminatePod)
	}
	if MsgTypeTerminalInput != "terminal_input" {
		t.Errorf("MsgTypeTerminalInput = %s, want terminal_input", MsgTypeTerminalInput)
	}
	if MsgTypeTerminalResize != "terminal_resize" {
		t.Errorf("MsgTypeTerminalResize = %s, want terminal_resize", MsgTypeTerminalResize)
	}
	if MsgTypeSendPrompt != "send_prompt" {
		t.Errorf("MsgTypeSendPrompt = %s, want send_prompt", MsgTypeSendPrompt)
	}
}

func TestDataStructs(t *testing.T) {
	t.Run("HeartbeatData", func(t *testing.T) {
		data := HeartbeatData{
			Pods:          []HeartbeatPod{{PodKey: "s1"}},
			RunnerVersion: "1.0.0",
		}
		if len(data.Pods) != 1 {
			t.Error("Pods not set correctly")
		}
	})

	t.Run("PodCreatedData", func(t *testing.T) {
		data := PodCreatedData{
			PodKey:       "s1",
			Pid:          12345,
			BranchName:   "main",
			WorktreePath: "/path/to/worktree",
			Cols:         80,
			Rows:         24,
		}
		if data.PodKey != "s1" || data.Pid != 12345 {
			t.Error("fields not set correctly")
		}
	})

	t.Run("PodTerminatedData", func(t *testing.T) {
		data := PodTerminatedData{PodKey: "s1", ExitCode: 1}
		if data.ExitCode != 1 {
			t.Error("ExitCode not set correctly")
		}
	})

	t.Run("TerminalOutputData", func(t *testing.T) {
		data := TerminalOutputData{PodKey: "s1", Data: []byte("output")}
		if string(data.Data) != "output" {
			t.Error("Data not set correctly")
		}
	})

	t.Run("AgentStatusData", func(t *testing.T) {
		data := AgentStatusData{PodKey: "s1", Status: "running", Pid: 123}
		if data.Status != "running" {
			t.Error("Status not set correctly")
		}
	})

	t.Run("PtyResizedData", func(t *testing.T) {
		data := PtyResizedData{PodKey: "s1", Cols: 80, Rows: 24}
		if data.Cols != 80 || data.Rows != 24 {
			t.Error("dimensions not set correctly")
		}
	})

	t.Run("CreatePodRequest", func(t *testing.T) {
		req := CreatePodRequest{
			PodKey:         "s1",
			InitialCommand: "claude",
			InitialPrompt:  "hello",
			PermissionMode: "plan",
			EnvVars:        map[string]string{"KEY": "VALUE"},
			PluginConfig: map[string]interface{}{
				"repository_url": "https://github.com/org/repo.git",
				"branch":         "main",
			},
		}
		if req.InitialCommand != "claude" {
			t.Error("InitialCommand not set correctly")
		}
		if req.PluginConfig["repository_url"] != "https://github.com/org/repo.git" {
			t.Error("PluginConfig repository_url not set correctly")
		}
	})

	t.Run("TerminalInputRequest", func(t *testing.T) {
		req := TerminalInputRequest{PodKey: "s1", Data: []byte("input")}
		if string(req.Data) != "input" {
			t.Error("Data not set correctly")
		}
	})

	t.Run("TerminalResizeRequest", func(t *testing.T) {
		req := TerminalResizeRequest{PodKey: "s1", Cols: 100, Rows: 50}
		if req.Cols != 100 || req.Rows != 50 {
			t.Error("dimensions not set correctly")
		}
	})
}

func TestErrorVariables(t *testing.T) {
	if ErrRunnerNotConnected == nil {
		t.Error("ErrRunnerNotConnected should not be nil")
	}
	if ErrRunnerNotConnected.Error() != "runner not connected" {
		t.Error("ErrRunnerNotConnected message incorrect")
	}

	if ErrConnectionClosed == nil {
		t.Error("ErrConnectionClosed should not be nil")
	}
	if ErrConnectionClosed.Error() != "connection closed" {
		t.Error("ErrConnectionClosed message incorrect")
	}
}

func TestConnectionManagerSharding(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	// Test that different runner IDs go to different shards
	shard1 := cm.getShard(1)
	shard2 := cm.getShard(257) // Should be in the same shard as 1 (1 % 256 == 257 % 256)
	shard3 := cm.getShard(2)   // Should be in a different shard

	if shard1 != shard2 {
		t.Error("runner 1 and 257 should be in the same shard")
	}
	if shard1 == shard3 {
		t.Error("runner 1 and 2 should be in different shards")
	}
}

func TestConnectionCount(t *testing.T) {
	cm := NewConnectionManager(newTestLogger())

	if cm.ConnectionCount() != 0 {
		t.Errorf("initial count should be 0, got %d", cm.ConnectionCount())
	}

	// Add some connections
	for i := int64(1); i <= 5; i++ {
		shard := cm.getShard(i)
		shard.mu.Lock()
		shard.connections[i] = &RunnerConnection{RunnerID: i, Send: make(chan []byte, 256)}
		cm.connCount.Add(1)
		shard.mu.Unlock()
	}

	if cm.ConnectionCount() != 5 {
		t.Errorf("count should be 5, got %d", cm.ConnectionCount())
	}
}
