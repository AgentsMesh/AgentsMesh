package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

// Terminate command backoff constants
const (
	terminateCooldown      = 5 * time.Minute  // 冷却期：两次 terminate 之间的最短间隔
	terminateCacheCleanup  = 30 * time.Minute // 缓存条目过期清理阈值
)

// PodCoordinator coordinates pod lifecycle events between backend and runners
type PodCoordinator struct {
	db                *gorm.DB
	connectionManager *RunnerConnectionManager
	terminalRouter    *TerminalRouter
	heartbeatBatcher  *HeartbeatBatcher
	logger            *slog.Logger

	// Command sender for sending commands to runners (e.g., GRPCCommandSender).
	// Defaults to NoOpCommandSender if not explicitly set via SetCommandSender.
	commandSender RunnerCommandSender

	// Relay connection cache for tracking Runner's relay connections
	relayConnectionCache *RelayConnectionCache

	// Terminate command backoff: prevents sending duplicate terminate commands
	// during heartbeat reconciliation. Maps podKey → last sent time.
	terminateSentCache map[string]time.Time
	terminateCacheMu   sync.Mutex

	// Callbacks
	onStatusChange   func(podKey string, status string, agentStatus string)
	onInitProgress   func(podKey string, phase string, progress int, message string)

	// Autopilot callbacks (set during initialization, read concurrently — safe because
	// they are set before any goroutine accesses them, establishing happens-before)
	onAutopilotStatusChange    AutopilotStatusChangeFunc
	onAutopilotIterationChange AutopilotIterationChangeFunc
	onAutopilotThinkingChange  AutopilotThinkingChangeFunc
}

// NewPodCoordinator creates a new pod coordinator.
// By default, uses NoOpCommandSender which logs warnings. Call SetCommandSender
// to configure a real command sender (e.g., GRPCCommandSender).
func NewPodCoordinator(
	db *gorm.DB,
	cm *RunnerConnectionManager,
	tr *TerminalRouter,
	hb *HeartbeatBatcher,
	logger *slog.Logger,
) *PodCoordinator {
	pc := &PodCoordinator{
		db:                   db,
		connectionManager:    cm,
		terminalRouter:       tr,
		heartbeatBatcher:     hb,
		logger:               logger,
		commandSender:        NewNoOpCommandSender(logger), // Default to no-op
		relayConnectionCache: NewRelayConnectionCache(),
		terminateSentCache:   make(map[string]time.Time),
	}

	// Set up callbacks from connection manager
	cm.SetHeartbeatCallback(pc.handleHeartbeat)
	cm.SetPodCreatedCallback(pc.handlePodCreated)
	cm.SetPodTerminatedCallback(pc.handlePodTerminated)
	cm.SetAgentStatusCallback(pc.handleAgentStatus)
	cm.SetPodErrorCallback(pc.handlePodError)
	cm.SetDisconnectCallback(pc.handleRunnerDisconnect)
	cm.SetPodInitProgressCallback(pc.handlePodInitProgress)

	// Set up AutopilotController callbacks
	cm.SetAutopilotStatusCallback(pc.handleAutopilotControllerStatus)
	cm.SetAutopilotCreatedCallback(pc.handleAutopilotControllerCreated)
	cm.SetAutopilotTerminatedCallback(pc.handleAutopilotControllerTerminated)
	cm.SetAutopilotIterationCallback(pc.handleAutopilotIteration)
	cm.SetAutopilotThinkingCallback(pc.handleAutopilotThinking)

	return pc
}

// SetCommandSender sets the command sender for sending commands to runners.
// This must be called before using CreatePod/TerminatePod methods.
func (pc *PodCoordinator) SetCommandSender(sender RunnerCommandSender) {
	pc.commandSender = sender
	pc.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
}

// GetCommandSender returns the command sender for sending commands to runners.
// Returns nil if no command sender is configured.
func (pc *PodCoordinator) GetCommandSender() RunnerCommandSender {
	return pc.commandSender
}

// SetStatusChangeCallback sets the callback for status changes
func (pc *PodCoordinator) SetStatusChangeCallback(fn func(podKey string, status string, agentStatus string)) {
	pc.onStatusChange = fn
}

// SetInitProgressCallback sets the callback for init progress events
func (pc *PodCoordinator) SetInitProgressCallback(fn func(podKey string, phase string, progress int, message string)) {
	pc.onInitProgress = fn
}

// IncrementPods increments pod count for a runner
func (pc *PodCoordinator) IncrementPods(ctx context.Context, runnerID int64) error {
	return pc.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = current_pods + 1 WHERE id = ?",
		runnerID,
	).Error
}

// DecrementPods decrements pod count for a runner
func (pc *PodCoordinator) DecrementPods(ctx context.Context, runnerID int64) error {
	return pc.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?",
		runnerID,
	).Error
}

// CreatePod creates a new pod on a runner
// Uses Proto type directly for zero-copy message passing.
func (pc *PodCoordinator) CreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	// Increment pod count first
	if err := pc.IncrementPods(ctx, runnerID); err != nil {
		return err
	}

	// Note: Pod is NOT registered with terminal router here.
	// Registration happens in handlePodCreated when Runner confirms the pod is actually created.
	// This ensures we don't have stale routes if pod creation fails on Runner side.

	// Send create pod command to runner via command sender.
	// Rollback pod count on failure to keep capacity accurate.
	if err := pc.commandSender.SendCreatePod(ctx, runnerID, cmd); err != nil {
		_ = pc.DecrementPods(ctx, runnerID)
		return err
	}
	return nil
}

// TerminatePod terminates a pod on a runner
func (pc *PodCoordinator) TerminatePod(ctx context.Context, podKey string) error {
	// Get pod to find runner
	var pod agentpod.Pod
	if err := pc.db.WithContext(ctx).
		Where("pod_key = ?", podKey).
		First(&pod).Error; err != nil {
		return err
	}

	// Send terminate request to runner via command sender
	// NoOpCommandSender will log warning if not configured
	if err := pc.commandSender.SendTerminatePod(ctx, pod.RunnerID, podKey); err != nil {
		pc.logger.Warn("failed to send terminate to runner, marking as completed",
			"pod_key", podKey,
			"error", err)
	}

	// Update pod status
	now := time.Now()
	if err := pc.db.WithContext(ctx).
		Model(&pod).
		Updates(map[string]interface{}{
			"status":      agentpod.StatusCompleted,
			"finished_at": now,
		}).Error; err != nil {
		return err
	}

	// Unregister from terminal router
	pc.terminalRouter.UnregisterPod(podKey)

	// Decrement pod count
	return pc.DecrementPods(ctx, pod.RunnerID)
}

// UpdateActivity updates last activity timestamp for a pod
func (pc *PodCoordinator) UpdateActivity(ctx context.Context, podKey string) error {
	return pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("last_activity", time.Now()).Error
}

// MarkDisconnected marks a pod as disconnected (user closed browser)
func (pc *PodCoordinator) MarkDisconnected(ctx context.Context, podKey string) error {
	return pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.StatusRunning).
		Update("status", agentpod.StatusDisconnected).Error
}

// MarkReconnected marks a pod as running again (user reconnected)
func (pc *PodCoordinator) MarkReconnected(ctx context.Context, podKey string) error {
	return pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.StatusDisconnected).
		Updates(map[string]interface{}{
			"status":        agentpod.StatusRunning,
			"last_activity": time.Now(),
		}).Error
}

// ==================== AutopilotController Commands ====================

// SendCreateAutopilot sends a create AutopilotController command to a runner.
// Delegates to the underlying command sender.
func (pc *PodCoordinator) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	return pc.commandSender.SendCreateAutopilot(runnerID, cmd)
}

// SendAutopilotControl sends an AutopilotController control command to a runner.
// Delegates to the underlying command sender.
func (pc *PodCoordinator) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	return pc.commandSender.SendAutopilotControl(runnerID, cmd)
}

// ==================== Terminate Command Backoff ====================

// isTerminateCooldown checks whether a terminate command for the given podKey
// is within the cooldown period.
func (pc *PodCoordinator) isTerminateCooldown(podKey string) bool {
	pc.terminateCacheMu.Lock()
	defer pc.terminateCacheMu.Unlock()
	if lastSent, ok := pc.terminateSentCache[podKey]; ok {
		return time.Since(lastSent) < terminateCooldown
	}
	return false
}

// recordTerminateSent records that a terminate command was sent for the given podKey.
// Also performs lazy cleanup of expired entries.
func (pc *PodCoordinator) recordTerminateSent(podKey string) {
	pc.terminateCacheMu.Lock()
	defer pc.terminateCacheMu.Unlock()
	now := time.Now()
	pc.terminateSentCache[podKey] = now

	// Lazy cleanup: remove entries older than terminateCacheCleanup
	for key, t := range pc.terminateSentCache {
		if now.Sub(t) > terminateCacheCleanup {
			delete(pc.terminateSentCache, key)
		}
	}
}

// ==================== Relay Connection Queries ====================

// GetRelayConnections returns relay connections for a runner
func (pc *PodCoordinator) GetRelayConnections(runnerID int64) []RelayConnectionInfo {
	return pc.relayConnectionCache.Get(runnerID)
}
