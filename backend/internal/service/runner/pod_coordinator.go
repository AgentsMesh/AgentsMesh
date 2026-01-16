package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

// PodCoordinator coordinates pod lifecycle events between backend and runners
type PodCoordinator struct {
	db                *gorm.DB
	connectionManager *ConnectionManager
	terminalRouter    *TerminalRouter
	heartbeatBatcher  *HeartbeatBatcher
	logger            *slog.Logger

	// Callbacks
	onStatusChange func(podKey string, status string, agentStatus string)
}

// NewPodCoordinator creates a new pod coordinator
func NewPodCoordinator(
	db *gorm.DB,
	cm *ConnectionManager,
	tr *TerminalRouter,
	hb *HeartbeatBatcher,
	logger *slog.Logger,
) *PodCoordinator {
	pc := &PodCoordinator{
		db:                db,
		connectionManager: cm,
		terminalRouter:    tr,
		heartbeatBatcher:  hb,
		logger:            logger,
	}

	// Set up callbacks from connection manager
	cm.SetHeartbeatCallback(pc.handleHeartbeat)
	cm.SetPodCreatedCallback(pc.handlePodCreated)
	cm.SetPodTerminatedCallback(pc.handlePodTerminated)
	cm.SetAgentStatusCallback(pc.handleAgentStatus)
	cm.SetDisconnectCallback(pc.handleRunnerDisconnect)

	return pc
}

// SetStatusChangeCallback sets the callback for status changes
func (pc *PodCoordinator) SetStatusChangeCallback(fn func(podKey string, status string, agentStatus string)) {
	pc.onStatusChange = fn
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
func (pc *PodCoordinator) CreatePod(ctx context.Context, runnerID int64, req *CreatePodRequest) error {
	// Increment pod count first
	if err := pc.IncrementPods(ctx, runnerID); err != nil {
		return err
	}

	// Register with terminal router
	pc.terminalRouter.RegisterPod(req.PodKey, runnerID)

	// Send create pod request to runner
	return pc.connectionManager.SendCreatePod(ctx, runnerID, req)
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

	// Send terminate request to runner
	if err := pc.connectionManager.SendTerminatePod(ctx, pod.RunnerID, podKey); err != nil {
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
