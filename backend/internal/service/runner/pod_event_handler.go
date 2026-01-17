package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleHeartbeat handles heartbeat from a runner (Proto type)
// Heartbeats are batched via HeartbeatBatcher for high-scale performance
func (pc *PodCoordinator) handleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	ctx := context.Background()

	// Record heartbeat via batcher (batched DB writes + immediate Redis update)
	// Note: RunnerVersion not available in Proto, using empty string
	if err := pc.heartbeatBatcher.RecordHeartbeat(
		ctx,
		runnerID,
		len(data.Pods),
		"online",
		"", // RunnerVersion not in Proto HeartbeatData
	); err != nil {
		pc.logger.Error("failed to record heartbeat",
			"runner_id", runnerID,
			"error", err)
	}

	// Reconcile pods
	reportedPodKeys := make(map[string]bool)
	for _, p := range data.Pods {
		reportedPodKeys[p.PodKey] = true
	}

	pc.reconcilePods(ctx, runnerID, reportedPodKeys)
}

// reconcilePods syncs pod status between runner heartbeat and database
func (pc *PodCoordinator) reconcilePods(ctx context.Context, runnerID int64, reportedPods map[string]bool) {
	now := time.Now()

	// First, ensure all reported pods are registered with terminal router
	// and restore any orphaned pods that runner reports as active
	for podKey := range reportedPods {
		// Always register pod with terminal router (idempotent operation)
		// This ensures routing works even after backend restart
		pc.terminalRouter.RegisterPod(podKey, runnerID)

		// Try to restore if pod is orphaned
		result := pc.db.WithContext(ctx).
			Model(&agentpod.Pod{}).
			Where("pod_key = ? AND runner_id = ? AND status = ?", podKey, runnerID, agentpod.StatusOrphaned).
			Updates(map[string]interface{}{
				"status":        agentpod.StatusRunning,
				"finished_at":   nil,
				"last_activity": now,
			})
		if result.Error != nil {
			pc.logger.Error("failed to restore orphaned pod",
				"pod_key", podKey,
				"error", result.Error)
		} else if result.RowsAffected > 0 {
			pc.logger.Info("restored orphaned pod reported by runner",
				"pod_key", podKey,
				"runner_id", runnerID)
		}
	}

	// Get active pods for this runner from database
	var pods []agentpod.Pod
	if err := pc.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{agentpod.StatusRunning, agentpod.StatusInitializing}).
		Find(&pods).Error; err != nil {
		pc.logger.Error("failed to get pods for reconciliation",
			"runner_id", runnerID,
			"error", err)
		return
	}

	// Mark pods that are in DB but not reported by runner as orphaned
	for _, p := range pods {
		if !reportedPods[p.PodKey] {
			if err := pc.db.WithContext(ctx).
				Model(&p).
				Updates(map[string]interface{}{
					"status":      agentpod.StatusOrphaned,
					"finished_at": now,
				}).Error; err != nil {
				pc.logger.Error("failed to mark pod as orphaned",
					"pod_key", p.PodKey,
					"error", err)
			} else {
				pc.logger.Warn("pod marked as orphaned (not reported by runner)",
					"pod_key", p.PodKey,
					"runner_id", runnerID)
				// Unregister from terminal router
				pc.terminalRouter.UnregisterPod(p.PodKey)
			}
		}
	}
}

// handlePodCreated handles pod creation event from runner (Proto type)
func (pc *PodCoordinator) handlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	ctx := context.Background()

	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       int(data.Pid),
		"status":        agentpod.StatusRunning,
		"started_at":    now,
		"last_activity": now,
	}

	// Note: BranchName and WorktreePath not available in Proto PodCreatedEvent

	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", data.PodKey).
		Updates(updates).Error; err != nil {
		pc.logger.Error("failed to update pod on creation",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	// Register with terminal router
	pc.terminalRouter.RegisterPod(data.PodKey, runnerID)

	pc.logger.Info("pod created",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"pid", data.Pid)

	// Notify status change
	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, agentpod.StatusRunning, "")
	}
}

// handlePodTerminated handles pod termination event from runner (Proto type)
func (pc *PodCoordinator) handlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	ctx := context.Background()

	now := time.Now()
	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", data.PodKey).
		Updates(map[string]interface{}{
			"status":      agentpod.StatusCompleted,
			"finished_at": now,
			"pty_pid":     nil,
		}).Error; err != nil {
		pc.logger.Error("failed to update pod on termination",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	// Decrement runner pod count
	pc.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?",
		runnerID,
	)

	// Unregister from terminal router
	pc.terminalRouter.UnregisterPod(data.PodKey)

	pc.logger.Info("pod terminated",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"exit_code", data.ExitCode)

	// Notify status change
	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, agentpod.StatusCompleted, "")
	}
}

// handleAgentStatus handles agent status change from runner (Proto type)
func (pc *PodCoordinator) handleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	ctx := context.Background()

	updates := map[string]interface{}{
		"agent_status": data.Status,
	}
	// Note: Pid not available in Proto AgentStatusEvent

	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Where("pod_key = ?", data.PodKey).
		Updates(updates).Error; err != nil {
		pc.logger.Error("failed to update agent status",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	pc.logger.Debug("agent status changed",
		"pod_key", data.PodKey,
		"status", data.Status)

	// Notify status change
	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, "", data.Status)
	}
}

// handleRunnerDisconnect handles runner disconnection
func (pc *PodCoordinator) handleRunnerDisconnect(runnerID int64) {
	ctx := context.Background()

	// Mark runner as offline, but don't immediately orphan pods
	// Pods will be orphaned by reconcilePods if runner doesn't reconnect
	// and report them in heartbeat
	if err := pc.db.WithContext(ctx).
		Table("runners").
		Where("id = ?", runnerID).
		Update("status", "offline").Error; err != nil {
		pc.logger.Error("failed to mark runner as offline",
			"runner_id", runnerID,
			"error", err)
	}

	pc.logger.Info("runner disconnected, pods will be reconciled on reconnect",
		"runner_id", runnerID)

	// Note: We intentionally don't mark pods as orphaned here
	// The runner might reconnect quickly (network glitch) and pods are still running
	// Pods will be properly reconciled when:
	// 1. Runner reconnects and sends heartbeat - reconcilePods will handle it
	// 2. Pod cleanup task runs and finds stale pods
}
