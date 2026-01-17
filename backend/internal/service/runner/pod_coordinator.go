package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
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

	// Callbacks
	onStatusChange func(podKey string, status string, agentStatus string)
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
		db:                db,
		connectionManager: cm,
		terminalRouter:    tr,
		heartbeatBatcher:  hb,
		logger:            logger,
		commandSender:     NewNoOpCommandSender(logger), // Default to no-op
	}

	// Set up callbacks from connection manager
	cm.SetHeartbeatCallback(pc.handleHeartbeat)
	cm.SetPodCreatedCallback(pc.handlePodCreated)
	cm.SetPodTerminatedCallback(pc.handlePodTerminated)
	cm.SetAgentStatusCallback(pc.handleAgentStatus)
	cm.SetDisconnectCallback(pc.handleRunnerDisconnect)

	return pc
}

// SetCommandSender sets the command sender for sending commands to runners.
// This must be called before using CreatePod/TerminatePod methods.
func (pc *PodCoordinator) SetCommandSender(sender RunnerCommandSender) {
	pc.commandSender = sender
	pc.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
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

	// Note: Pod is NOT registered with terminal router here.
	// Registration happens in handlePodCreated when Runner confirms the pod is actually created.
	// This ensures we don't have stale routes if pod creation fails on Runner side.

	// Send create pod request to runner via command sender
	return pc.commandSender.SendCreatePod(ctx, runnerID, req)
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
