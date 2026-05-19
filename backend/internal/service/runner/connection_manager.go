package runner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	otelinit "github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const numShards = 256

const DefaultInitTimeout = 30 * time.Second

type RunnerConnectionManager struct {
	shards            [numShards]*grpcConnectionShard
	logger            *slog.Logger
	connCount         atomic.Int64
	generationCounter atomic.Int64 // Monotonic counter for connection generation IDs
	pingInterval      time.Duration

	initTimeout     time.Duration
	initTimeoutStop chan struct{}
	initTimeoutOnce sync.Once

	agentsProvider interfaces.AgentsProvider
	serverVersion      string

	onHeartbeat          func(runnerID int64, data *runnerv1.HeartbeatData)
	onPodCreated         func(runnerID int64, data *runnerv1.PodCreatedEvent)
	onPodTerminated      func(runnerID int64, data *runnerv1.PodTerminatedEvent)
	onAgentStatus        func(runnerID int64, data *runnerv1.AgentStatusEvent)
	onPodInitProgress    func(runnerID int64, data *runnerv1.PodInitProgressEvent)
	onRequestRelayToken  func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)
	onDisconnect         func(runnerID int64)
	onInitialized        func(runnerID int64, availableAgents []string)
	onInitFailed         func(runnerID int64, reason string)
	onSandboxesStatus    func(runnerID int64, data *runnerv1.SandboxesStatusEvent)
	onOSCNotification    func(runnerID int64, data *runnerv1.OSCNotificationEvent)
	onPodError           func(runnerID int64, data *runnerv1.ErrorEvent)
	onOSCTitle           func(runnerID int64, data *runnerv1.OSCTitleEvent)

	onObservePodResult func(runnerID int64, data *runnerv1.ObservePodResult)

	onUpgradeStatus func(runnerID int64, data *runnerv1.UpgradeStatusEvent)

	onLogUploadStatus func(runnerID int64, data *runnerv1.LogUploadStatusEvent)

	onPodRestarting func(runnerID int64, data *runnerv1.PodRestartingEvent)

	onAutopilotStatus     func(runnerID int64, data *runnerv1.AutopilotStatusEvent)
	onAutopilotIteration  func(runnerID int64, data *runnerv1.AutopilotIterationEvent)
	onAutopilotCreated    func(runnerID int64, data *runnerv1.AutopilotCreatedEvent)
	onAutopilotTerminated func(runnerID int64, data *runnerv1.AutopilotTerminatedEvent)
	onAutopilotThinking   func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)

	onTokenUsage func(runnerID int64, data *runnerv1.TokenUsageReport)
}

type grpcConnectionShard struct {
	connections map[int64]*GRPCConnection
	mu          sync.RWMutex
}

func NewRunnerConnectionManager(logger *slog.Logger) *RunnerConnectionManager {
	cm := &RunnerConnectionManager{
		logger:          logger,
		pingInterval:    30 * time.Second,
		initTimeout:     DefaultInitTimeout,
		initTimeoutStop: make(chan struct{}),
	}

	for i := 0; i < numShards; i++ {
		cm.shards[i] = &grpcConnectionShard{
			connections: make(map[int64]*GRPCConnection),
		}
	}

	return cm
}

func (cm *RunnerConnectionManager) getShard(runnerID int64) *grpcConnectionShard {
	idx := uint64(runnerID) % numShards
	return cm.shards[idx]
}

func (cm *RunnerConnectionManager) AddConnection(runnerID int64, nodeID, orgSlug string, stream RunnerStream) *GRPCConnection {
	shard := cm.getShard(runnerID)
	gen := cm.generationCounter.Add(1)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if existing, ok := shard.connections[runnerID]; ok {
		existing.Close()
		cm.connCount.Add(-1)
	}

	conn := NewGRPCConnection(runnerID, gen, nodeID, orgSlug, stream)
	shard.connections[runnerID] = conn
	cm.connCount.Add(1)
	otelinit.RunnerConnected.Add(context.Background(), 1)

	cm.logger.Info("runner connected (gRPC)",
		"runner_id", runnerID,
		"node_id", nodeID,
		"org_slug", orgSlug,
		"generation", gen,
		"total_connections", cm.connCount.Load(),
	)

	return conn
}

func (cm *RunnerConnectionManager) RemoveConnection(runnerID, generation int64) {
	shard := cm.getShard(runnerID)

	shard.mu.Lock()
	conn, ok := shard.connections[runnerID]
	if ok && conn.Generation != generation {
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
		otelinit.RunnerConnected.Add(context.Background(), -1)
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

func (cm *RunnerConnectionManager) GetConnection(runnerID int64) *GRPCConnection {
	shard := cm.getShard(runnerID)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return shard.connections[runnerID]
}

func (cm *RunnerConnectionManager) IsConnected(runnerID int64) bool {
	return cm.GetConnection(runnerID) != nil
}

func (cm *RunnerConnectionManager) UpdateHeartbeat(runnerID int64) {
	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.UpdateLastPing()
	}
}

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

func (cm *RunnerConnectionManager) ConnectionCount() int64 {
	return cm.connCount.Load()
}

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
