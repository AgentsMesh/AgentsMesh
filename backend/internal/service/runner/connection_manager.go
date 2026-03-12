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
	shards            [numShards]*grpcConnectionShard
	logger            *slog.Logger
	connCount         atomic.Int64
	generationCounter atomic.Int64 // Monotonic counter for connection generation IDs
	pingInterval      time.Duration

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
	onAgentStatus        func(runnerID int64, data *runnerv1.AgentStatusEvent)
	onPtyResized         func(runnerID int64, data *runnerv1.PtyResizedEvent)
	onPodInitProgress    func(runnerID int64, data *runnerv1.PodInitProgressEvent)
	onRequestRelayToken  func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)
	onDisconnect         func(runnerID int64)
	onInitialized        func(runnerID int64, availableAgents []string)
	onInitFailed         func(runnerID int64, reason string)
	onSandboxesStatus    func(runnerID int64, data *runnerv1.SandboxesStatusEvent)
	onOSCNotification    func(runnerID int64, data *runnerv1.OSCNotificationEvent)
	onPodError           func(runnerID int64, data *runnerv1.ErrorEvent)
	onOSCTitle           func(runnerID int64, data *runnerv1.OSCTitleEvent)

	// Upgrade event callback
	onUpgradeStatus func(runnerID int64, data *runnerv1.UpgradeStatusEvent)

	// Log upload event callback
	onLogUploadStatus func(runnerID int64, data *runnerv1.LogUploadStatusEvent)

	// AutopilotController event callbacks
	onAutopilotStatus     func(runnerID int64, data *runnerv1.AutopilotStatusEvent)
	onAutopilotIteration  func(runnerID int64, data *runnerv1.AutopilotIterationEvent)
	onAutopilotCreated    func(runnerID int64, data *runnerv1.AutopilotCreatedEvent)
	onAutopilotTerminated func(runnerID int64, data *runnerv1.AutopilotTerminatedEvent)
	onAutopilotThinking   func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)

	// Token usage callback
	onTokenUsage func(runnerID int64, data *runnerv1.TokenUsageReport)
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

// ==================== Connection Management ====================

// AddConnection adds a gRPC connection.
// Returns the new connection with a unique generation ID.
func (cm *RunnerConnectionManager) AddConnection(runnerID int64, nodeID, orgSlug string, stream RunnerStream) *GRPCConnection {
	shard := cm.getShard(runnerID)
	gen := cm.generationCounter.Add(1)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Close existing connection if any
	if existing, ok := shard.connections[runnerID]; ok {
		existing.Close()
		cm.connCount.Add(-1)
	}

	conn := NewGRPCConnection(runnerID, gen, nodeID, orgSlug, stream)
	shard.connections[runnerID] = conn
	cm.connCount.Add(1)

	cm.logger.Info("runner connected (gRPC)",
		"runner_id", runnerID,
		"node_id", nodeID,
		"org_slug", orgSlug,
		"generation", gen,
		"total_connections", cm.connCount.Load(),
	)

	return conn
}

// RemoveConnection removes a gRPC connection only if the generation matches.
// If the current connection has a different generation (newer connection replaced it),
// the removal is skipped to prevent stale defer calls from closing active connections.
func (cm *RunnerConnectionManager) RemoveConnection(runnerID, generation int64) {
	shard := cm.getShard(runnerID)

	shard.mu.Lock()
	conn, ok := shard.connections[runnerID]
	if ok && conn.Generation != generation {
		// Generation mismatch: a newer connection has replaced this one.
		// Skip removal to avoid closing the active connection.
		shard.mu.Unlock()
		cm.logger.Debug("generation mismatch, skipping remove",
			"runner_id", runnerID,
			"remove_generation", generation,
			"current_generation", conn.Generation,
		)
		return
	}
	if ok {
		delete(shard.connections, runnerID)
		cm.connCount.Add(-1)
	}
	shard.mu.Unlock()

	if ok {
		conn.Close()
		cm.logger.Info("runner disconnected (gRPC)",
			"runner_id", runnerID,
			"generation", generation,
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
	type timedOutEntry struct {
		runnerID   int64
		generation int64
	}
	var timedOut []timedOutEntry

	for i := 0; i < numShards; i++ {
		shard := cm.shards[i]
		shard.mu.RLock()
		for runnerID, conn := range shard.connections {
			if !conn.IsInitialized() && now.Sub(conn.ConnectedAt) > cm.initTimeout {
				timedOut = append(timedOut, timedOutEntry{runnerID: runnerID, generation: conn.Generation})
			}
		}
		shard.mu.RUnlock()
	}

	for _, entry := range timedOut {
		reason := "initialization timeout"
		cm.logger.Warn("removing gRPC connection due to initialization timeout",
			"runner_id", entry.runnerID,
			"timeout", cm.initTimeout,
		)

		if cm.onInitFailed != nil {
			cm.onInitFailed(entry.runnerID, reason)
		}

		cm.RemoveConnection(entry.runnerID, entry.generation)
	}
}
