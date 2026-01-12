package runner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// numShards is the number of shards for connection partitioning
// 256 shards reduce lock contention by ~256x for 100K runners
const numShards = 256

// connectionShard holds a subset of connections with its own lock
type connectionShard struct {
	connections map[int64]*RunnerConnection
	mu          sync.RWMutex
}

// ConnectionManager manages runner WebSocket connections using sharded locks
type ConnectionManager struct {
	shards       [numShards]*connectionShard
	logger       *slog.Logger
	pingInterval time.Duration
	pingTimeout  time.Duration
	connCount    atomic.Int64 // Total connection count for metrics

	// Event callbacks
	onHeartbeat      func(runnerID int64, data *HeartbeatData)
	onPodCreated     func(runnerID int64, data *PodCreatedData)
	onPodTerminated  func(runnerID int64, data *PodTerminatedData)
	onTerminalOutput func(runnerID int64, data *TerminalOutputData)
	onAgentStatus    func(runnerID int64, data *AgentStatusData)
	onPtyResized     func(runnerID int64, data *PtyResizedData)
	onDisconnect     func(runnerID int64)
}

// NewConnectionManager creates a new connection manager with sharded locks
func NewConnectionManager(logger *slog.Logger) *ConnectionManager {
	cm := &ConnectionManager{
		logger:       logger,
		pingInterval: 30 * time.Second,
		pingTimeout:  60 * time.Second,
	}

	// Initialize all shards
	for i := 0; i < numShards; i++ {
		cm.shards[i] = &connectionShard{
			connections: make(map[int64]*RunnerConnection),
		}
	}

	return cm
}

// getShard returns the shard for a given runner ID using modulo hashing
func (cm *ConnectionManager) getShard(runnerID int64) *connectionShard {
	// Use unsigned modulo to ensure positive index
	idx := uint64(runnerID) % numShards
	return cm.shards[idx]
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
	shard := cm.getShard(runnerID)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Close existing connection if any
	if existing, ok := shard.connections[runnerID]; ok {
		existing.Close()
		cm.connCount.Add(-1)
	}

	rc := &RunnerConnection{
		RunnerID: runnerID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
	}

	shard.connections[runnerID] = rc
	cm.connCount.Add(1)
	cm.logger.Info("runner connected", "runner_id", runnerID, "total_connections", cm.connCount.Load())

	return rc
}

// RemoveConnection removes a runner connection
func (cm *ConnectionManager) RemoveConnection(runnerID int64) {
	shard := cm.getShard(runnerID)

	shard.mu.Lock()
	conn, ok := shard.connections[runnerID]
	if ok {
		delete(shard.connections, runnerID)
		cm.connCount.Add(-1)
	}
	shard.mu.Unlock()

	if ok {
		conn.Close()
		cm.logger.Info("runner disconnected", "runner_id", runnerID, "total_connections", cm.connCount.Load())

		if cm.onDisconnect != nil {
			cm.onDisconnect(runnerID)
		}
	}
}

// GetConnection returns a runner connection
func (cm *ConnectionManager) GetConnection(runnerID int64) *RunnerConnection {
	shard := cm.getShard(runnerID)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return shard.connections[runnerID]
}

// IsConnected checks if a runner is connected
func (cm *ConnectionManager) IsConnected(runnerID int64) bool {
	shard := cm.getShard(runnerID)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	_, ok := shard.connections[runnerID]
	return ok
}

// UpdateHeartbeat updates the last ping time for a runner
func (cm *ConnectionManager) UpdateHeartbeat(runnerID int64) {
	shard := cm.getShard(runnerID)

	shard.mu.RLock()
	conn, ok := shard.connections[runnerID]
	shard.mu.RUnlock()

	if ok {
		conn.mu.Lock()
		conn.LastPing = time.Now()
		conn.mu.Unlock()
	}
}

// GetConnectedRunnerIDs returns IDs of all connected runners
// Note: This operation iterates all shards and should be used sparingly
func (cm *ConnectionManager) GetConnectedRunnerIDs() []int64 {
	// Pre-allocate based on atomic counter for efficiency
	ids := make([]int64, 0, cm.connCount.Load())

	for i := 0; i < numShards; i++ {
		shard := cm.shards[i]
		shard.mu.RLock()
		for id := range shard.connections {
			ids = append(ids, id)
		}
		shard.mu.RUnlock()
	}
	return ids
}

// ConnectionCount returns the total number of active connections
func (cm *ConnectionManager) ConnectionCount() int64 {
	return cm.connCount.Load()
}

// Close closes the connection manager and all connections
func (cm *ConnectionManager) Close() {
	for i := 0; i < numShards; i++ {
		shard := cm.shards[i]
		shard.mu.Lock()
		for _, conn := range shard.connections {
			conn.Close()
		}
		shard.connections = make(map[int64]*RunnerConnection)
		shard.mu.Unlock()
	}
	cm.connCount.Store(0)
}

// ========== Send Operations ==========
// These methods delegate to RunnerSender for actual implementation

// SendMessage sends a message to a runner
func (cm *ConnectionManager) SendMessage(ctx context.Context, runnerID int64, msg *RunnerMessage) error {
	return NewRunnerSender(cm).SendMessage(ctx, runnerID, msg)
}

// SendCreatePod sends a create pod request to a runner
func (cm *ConnectionManager) SendCreatePod(ctx context.Context, runnerID int64, req *CreatePodRequest) error {
	return NewRunnerSender(cm).SendCreatePod(ctx, runnerID, req)
}

// SendTerminatePod sends a terminate pod request to a runner
func (cm *ConnectionManager) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	return NewRunnerSender(cm).SendTerminatePod(ctx, runnerID, podKey)
}

// SendTerminalInput sends terminal input to a runner
func (cm *ConnectionManager) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	return NewRunnerSender(cm).SendTerminalInput(ctx, runnerID, podKey, data)
}

// SendTerminalResize sends terminal resize to a runner
func (cm *ConnectionManager) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	return NewRunnerSender(cm).SendTerminalResize(ctx, runnerID, podKey, cols, rows)
}

// SendPrompt sends a prompt to a pod
func (cm *ConnectionManager) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	return NewRunnerSender(cm).SendPrompt(ctx, runnerID, podKey, prompt)
}
