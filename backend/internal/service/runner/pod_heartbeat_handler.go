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

	// Update relay connection cache
	if data.RelayConnections != nil {
		connections := make([]RelayConnectionInfo, 0, len(data.RelayConnections))
		for _, rc := range data.RelayConnections {
			connections = append(connections, RelayConnectionInfo{
				PodKey:      rc.PodKey,
				RelayURL:    rc.RelayUrl,
				SessionID:   rc.SessionId,
				Connected:   rc.Connected,
				ConnectedAt: time.UnixMilli(rc.ConnectedAt),
			})
		}
		pc.relayConnectionCache.Update(runnerID, connections)
	}

	// Reconcile pods
	reportedPodKeys := make(map[string]bool)
	for _, p := range data.Pods {
		reportedPodKeys[p.PodKey] = true
		// Silently sync agent_status from heartbeat (no WebSocket event)
		if p.AgentStatus != "" {
			pc.db.WithContext(ctx).Model(&agentpod.Pod{}).
				Where("pod_key = ?", p.PodKey).
				Update("agent_status", p.AgentStatus)
		}
	}

	pc.reconcilePods(ctx, runnerID, reportedPodKeys)
}

// reconcilePods syncs pod status between runner heartbeat and database
func (pc *PodCoordinator) reconcilePods(ctx context.Context, runnerID int64, reportedPods map[string]bool) {
	now := time.Now()

	// First, check reported pods against database status
	// - Restore orphaned pods that runner reports as active
	// - Terminate pods that should not be running (terminated/completed in DB)
	for podKey := range reportedPods {
		var pod agentpod.Pod
		err := pc.db.WithContext(ctx).
			Where("pod_key = ? AND runner_id = ?", podKey, runnerID).
			First(&pod).Error

		if err != nil {
			// Pod not found in database, tell runner to terminate it
			pc.logger.Warn("runner reported unknown pod, sending terminate",
				"pod_key", podKey,
				"runner_id", runnerID)
			if sendErr := pc.commandSender.SendTerminatePod(ctx, runnerID, podKey); sendErr != nil {
				pc.logger.Error("failed to send terminate for unknown pod",
					"pod_key", podKey,
					"error", sendErr)
			}
			continue
		}

		// Check if pod should be terminated (already completed/terminated in DB)
		if pod.Status == agentpod.StatusCompleted || pod.Status == agentpod.StatusTerminated {
			pc.logger.Warn("runner reported terminated pod, sending terminate",
				"pod_key", podKey,
				"runner_id", runnerID,
				"db_status", pod.Status)
			if sendErr := pc.commandSender.SendTerminatePod(ctx, runnerID, podKey); sendErr != nil {
				pc.logger.Error("failed to send terminate for completed pod",
					"pod_key", podKey,
					"error", sendErr)
			}
			continue
		}

		// Ensure pod is registered with terminal router (preserves existing VT state)
		// This ensures routing works even after backend restart, without clearing terminal data
		pc.terminalRouter.EnsurePodRegistered(podKey, runnerID)

		// Try to restore if pod is orphaned
		if pod.Status == agentpod.StatusOrphaned {
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

				// Notify status change for WebSocket event
				if pc.onStatusChange != nil {
					pc.onStatusChange(podKey, agentpod.StatusRunning, "")
				}
			}
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
	// This means the pod process has terminated on the runner side
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

				// Notify status change for WebSocket event
				if pc.onStatusChange != nil {
					pc.onStatusChange(p.PodKey, agentpod.StatusOrphaned, "")
				}
			}
		}
	}

	// Update current_pods count based on actual running pods reported by runner
	// This ensures consistency between runner state and database
	activePodCount := 0
	for podKey := range reportedPods {
		// Only count pods that are actually active (not terminated/completed in DB)
		var pod agentpod.Pod
		if err := pc.db.WithContext(ctx).
			Where("pod_key = ? AND status IN ?", podKey, []string{agentpod.StatusRunning, agentpod.StatusInitializing}).
			First(&pod).Error; err == nil {
			activePodCount++
		}
	}

	// Update runner's current_pods to reflect actual active pods
	if err := pc.db.WithContext(ctx).
		Model(&agentpod.Pod{}).
		Table("runners").
		Where("id = ?", runnerID).
		Update("current_pods", activePodCount).Error; err != nil {
		pc.logger.Error("failed to update runner current_pods",
			"runner_id", runnerID,
			"active_pods", activePodCount,
			"error", err)
	}
}
