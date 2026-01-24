package runner

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// numShards is the number of shards for connection partitioning.
// 256 shards reduce lock contention by ~256x for 100K runners.
const numShards = 256

// DefaultInitTimeout is the default timeout for runner initialization.
const DefaultInitTimeout = 30 * time.Second

// RunnerConnectionManager manages runner connections using sharded locks.
//
// Architecture:
// - 256 shards to reduce lock contention for high concurrency (100K+ runners)
// - Connections are added when gRPC stream is established
// - Messages are sent through the connection's Send channel
// - Callbacks use Proto types directly (no JSON serialization overhead)
type RunnerConnectionManager struct {
	shards       [numShards]*grpcConnectionShard
	logger       *slog.Logger
	connCount    atomic.Int64
	pingInterval time.Duration

	// Initialization timeout
	initTimeout     time.Duration
	initTimeoutStop chan struct{}
	initTimeoutOnce sync.Once

	// Agent types provider and server version for initialization handshake
	agentTypesProvider interfaces.AgentTypesProvider
	serverVersion      string

	// Event callbacks - use Proto types directly for zero-copy efficiency
	onHeartbeat          func(runnerID int64, data *runnerv1.HeartbeatData)
	onPodCreated         func(runnerID int64, data *runnerv1.PodCreatedEvent)
	onPodTerminated      func(runnerID int64, data *runnerv1.PodTerminatedEvent)
	onTerminalOutput     func(runnerID int64, data *runnerv1.TerminalOutputEvent)
	onAgentStatus        func(runnerID int64, data *runnerv1.AgentStatusEvent)
	onPtyResized         func(runnerID int64, data *runnerv1.PtyResizedEvent)
	onPodInitProgress    func(runnerID int64, data *runnerv1.PodInitProgressEvent)
	onRequestRelayToken  func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)
	onDisconnect         func(runnerID int64)
	onInitialized        func(runnerID int64, availableAgents []string)
	onInitFailed         func(runnerID int64, reason string)
}

// grpcConnectionShard holds a subset of gRPC connections with its own lock.
type grpcConnectionShard struct {
	connections map[int64]*GRPCConnection
	mu          sync.RWMutex
}

// NewRunnerConnectionManager creates a new runner connection manager.
func NewRunnerConnectionManager(logger *slog.Logger) *RunnerConnectionManager {
	cm := &RunnerConnectionManager{
		logger:          logger,
		pingInterval:    30 * time.Second,
		initTimeout:     DefaultInitTimeout,
		initTimeoutStop: make(chan struct{}),
	}

	// Initialize all shards
	for i := 0; i < numShards; i++ {
		cm.shards[i] = &grpcConnectionShard{
			connections: make(map[int64]*GRPCConnection),
		}
	}

	return cm
}

// getShard returns the shard for a given runner ID.
func (cm *RunnerConnectionManager) getShard(runnerID int64) *grpcConnectionShard {
	idx := uint64(runnerID) % numShards
	return cm.shards[idx]
}

// ==================== Callback Setters ====================

// SetHeartbeatCallback sets the heartbeat callback (Proto type)
func (cm *RunnerConnectionManager) SetHeartbeatCallback(fn func(runnerID int64, data *runnerv1.HeartbeatData)) {
	cm.onHeartbeat = fn
}

// SetPodCreatedCallback sets the pod created callback (Proto type)
func (cm *RunnerConnectionManager) SetPodCreatedCallback(fn func(runnerID int64, data *runnerv1.PodCreatedEvent)) {
	cm.onPodCreated = fn
}

// SetPodTerminatedCallback sets the pod terminated callback (Proto type)
func (cm *RunnerConnectionManager) SetPodTerminatedCallback(fn func(runnerID int64, data *runnerv1.PodTerminatedEvent)) {
	cm.onPodTerminated = fn
}

// SetTerminalOutputCallback sets the terminal output callback (Proto type)
func (cm *RunnerConnectionManager) SetTerminalOutputCallback(fn func(runnerID int64, data *runnerv1.TerminalOutputEvent)) {
	cm.onTerminalOutput = fn
}

// SetAgentStatusCallback sets the agent status callback (Proto type)
func (cm *RunnerConnectionManager) SetAgentStatusCallback(fn func(runnerID int64, data *runnerv1.AgentStatusEvent)) {
	cm.onAgentStatus = fn
}

// SetPtyResizedCallback sets the PTY resized callback (Proto type)
func (cm *RunnerConnectionManager) SetPtyResizedCallback(fn func(runnerID int64, data *runnerv1.PtyResizedEvent)) {
	cm.onPtyResized = fn
}

// SetPodInitProgressCallback sets the pod init progress callback (Proto type)
func (cm *RunnerConnectionManager) SetPodInitProgressCallback(fn func(runnerID int64, data *runnerv1.PodInitProgressEvent)) {
	cm.onPodInitProgress = fn
}

// SetRequestRelayTokenCallback sets the request relay token callback (Proto type)
func (cm *RunnerConnectionManager) SetRequestRelayTokenCallback(fn func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)) {
	cm.onRequestRelayToken = fn
}

// SetDisconnectCallback sets the disconnect callback
func (cm *RunnerConnectionManager) SetDisconnectCallback(fn func(runnerID int64)) {
	cm.onDisconnect = fn
}

// SetInitializedCallback sets the initialized callback
func (cm *RunnerConnectionManager) SetInitializedCallback(fn func(runnerID int64, availableAgents []string)) {
	cm.onInitialized = fn
}

// SetInitFailedCallback sets the initialization failure callback
func (cm *RunnerConnectionManager) SetInitFailedCallback(fn func(runnerID int64, reason string)) {
	cm.onInitFailed = fn
}

// SetInitTimeout sets the initialization timeout
func (cm *RunnerConnectionManager) SetInitTimeout(timeout time.Duration) {
	cm.initTimeout = timeout
}

// SetPingInterval sets the ping interval
func (cm *RunnerConnectionManager) SetPingInterval(interval time.Duration) {
	cm.pingInterval = interval
}

// SetAgentTypesProvider sets the agent types provider for initialization handshake
func (cm *RunnerConnectionManager) SetAgentTypesProvider(provider interfaces.AgentTypesProvider) {
	cm.agentTypesProvider = provider
}

// SetServerVersion sets the server version for initialization handshake
func (cm *RunnerConnectionManager) SetServerVersion(version string) {
	cm.serverVersion = version
}

// GetHeartbeatCallback returns the current heartbeat callback
func (cm *RunnerConnectionManager) GetHeartbeatCallback() func(runnerID int64, data *runnerv1.HeartbeatData) {
	return cm.onHeartbeat
}

// GetDisconnectCallback returns the current disconnect callback
func (cm *RunnerConnectionManager) GetDisconnectCallback() func(runnerID int64) {
	return cm.onDisconnect
}

// ==================== Connection Management ====================

// AddConnection adds a gRPC connection.
func (cm *RunnerConnectionManager) AddConnection(runnerID int64, nodeID, orgSlug string, stream RunnerStream) *GRPCConnection {
	shard := cm.getShard(runnerID)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Close existing connection if any
	if existing, ok := shard.connections[runnerID]; ok {
		existing.Close()
		cm.connCount.Add(-1)
	}

	conn := NewGRPCConnection(runnerID, nodeID, orgSlug, stream)
	shard.connections[runnerID] = conn
	cm.connCount.Add(1)

	cm.logger.Info("runner connected (gRPC)",
		"runner_id", runnerID,
		"node_id", nodeID,
		"org_slug", orgSlug,
		"total_connections", cm.connCount.Load(),
	)

	return conn
}

// RemoveConnection removes a gRPC connection.
func (cm *RunnerConnectionManager) RemoveConnection(runnerID int64) {
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
		cm.logger.Info("runner disconnected (gRPC)",
			"runner_id", runnerID,
			"total_connections", cm.connCount.Load(),
		)

		if cm.onDisconnect != nil {
			cm.onDisconnect(runnerID)
		}
	}
}

// GetConnection returns a gRPC connection.
func (cm *RunnerConnectionManager) GetConnection(runnerID int64) *GRPCConnection {
	shard := cm.getShard(runnerID)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return shard.connections[runnerID]
}

// IsConnected checks if a runner is connected.
func (cm *RunnerConnectionManager) IsConnected(runnerID int64) bool {
	return cm.GetConnection(runnerID) != nil
}

// UpdateHeartbeat updates the last ping time for a runner.
func (cm *RunnerConnectionManager) UpdateHeartbeat(runnerID int64) {
	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.UpdateLastPing()
	}
}

// GetConnectedRunnerIDs returns IDs of all connected runners.
func (cm *RunnerConnectionManager) GetConnectedRunnerIDs() []int64 {
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

// ConnectionCount returns the total number of active connections.
func (cm *RunnerConnectionManager) ConnectionCount() int64 {
	return cm.connCount.Load()
}

// Close closes the connection manager and all connections.
func (cm *RunnerConnectionManager) Close() {
	cm.initTimeoutOnce.Do(func() {
		close(cm.initTimeoutStop)
	})

	for i := 0; i < numShards; i++ {
		shard := cm.shards[i]
		shard.mu.Lock()
		for _, conn := range shard.connections {
			conn.Close()
		}
		shard.connections = make(map[int64]*GRPCConnection)
		shard.mu.Unlock()
	}
	cm.connCount.Store(0)
}

// ==================== Initialization Timeout ====================

// StartInitTimeoutChecker starts the initialization timeout checker.
func (cm *RunnerConnectionManager) StartInitTimeoutChecker() {
	go cm.initTimeoutLoop()
}

func (cm *RunnerConnectionManager) initTimeoutLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.initTimeoutStop:
			return
		case <-ticker.C:
			cm.checkInitTimeouts()
		}
	}
}

func (cm *RunnerConnectionManager) checkInitTimeouts() {
	now := time.Now()
	var timedOutRunners []int64

	for i := 0; i < numShards; i++ {
		shard := cm.shards[i]
		shard.mu.RLock()
		for runnerID, conn := range shard.connections {
			if !conn.IsInitialized() && now.Sub(conn.ConnectedAt) > cm.initTimeout {
				timedOutRunners = append(timedOutRunners, runnerID)
			}
		}
		shard.mu.RUnlock()
	}

	for _, runnerID := range timedOutRunners {
		reason := "initialization timeout"
		cm.logger.Warn("removing gRPC connection due to initialization timeout",
			"runner_id", runnerID,
			"timeout", cm.initTimeout,
		)

		if cm.onInitFailed != nil {
			cm.onInitFailed(runnerID, reason)
		}

		cm.RemoveConnection(runnerID)
	}
}

// ==================== Proto Message Handlers (called by GRPCRunnerAdapter) ====================

// HandleHeartbeat handles heartbeat from a runner (Proto type)
func (cm *RunnerConnectionManager) HandleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onHeartbeat != nil {
		cm.onHeartbeat(runnerID, data)
	}
}

// HandlePodCreated handles pod created event (Proto type)
func (cm *RunnerConnectionManager) HandlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodCreated != nil {
		cm.onPodCreated(runnerID, data)
	}
}

// HandlePodTerminated handles pod terminated event (Proto type)
func (cm *RunnerConnectionManager) HandlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodTerminated != nil {
		cm.onPodTerminated(runnerID, data)
	}
}

// HandleTerminalOutput handles terminal output event (Proto type)
func (cm *RunnerConnectionManager) HandleTerminalOutput(runnerID int64, data *runnerv1.TerminalOutputEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onTerminalOutput != nil {
		cm.onTerminalOutput(runnerID, data)
	}
}

// HandleAgentStatus handles agent status event (Proto type)
func (cm *RunnerConnectionManager) HandleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onAgentStatus != nil {
		cm.onAgentStatus(runnerID, data)
	}
}

// HandlePtyResized handles PTY resized event (Proto type)
func (cm *RunnerConnectionManager) HandlePtyResized(runnerID int64, data *runnerv1.PtyResizedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPtyResized != nil {
		cm.onPtyResized(runnerID, data)
	}
}

// HandlePodInitProgress handles pod init progress event (Proto type)
func (cm *RunnerConnectionManager) HandlePodInitProgress(runnerID int64, data *runnerv1.PodInitProgressEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodInitProgress != nil {
		cm.onPodInitProgress(runnerID, data)
	}
}

// HandleInitialized handles initialized confirmation (Proto type)
func (cm *RunnerConnectionManager) HandleInitialized(runnerID int64, availableAgents []string) {
	cm.UpdateHeartbeat(runnerID)

	// Mark connection as initialized
	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.SetInitialized(true, availableAgents)
	}

	if cm.onInitialized != nil {
		cm.onInitialized(runnerID, availableAgents)
	}
}

// HandleRequestRelayToken handles relay token refresh request (Proto type)
func (cm *RunnerConnectionManager) HandleRequestRelayToken(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onRequestRelayToken != nil {
		cm.onRequestRelayToken(runnerID, data)
	}
}
