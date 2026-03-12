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
			_ = pc.podRepo.UpdateField(ctx, p.PodKey, "agent_status", p.AgentStatus)
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
		pod, err := pc.podRepo.GetByKeyAndRunner(ctx, podKey, runnerID)
		if err != nil {
			// Pod not found in database, tell runner to terminate it
			if pc.isTerminateCooldown(podKey) {
				continue
			}
			pc.recordTerminateSent(podKey)
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
			// Apply cooldown to prevent sending terminate every heartbeat cycle
			if pc.isTerminateCooldown(podKey) {
				pc.logger.Debug("terminate cooldown active, skipping",
					"pod_key", podKey,
					"runner_id", runnerID)
				continue
			}
			pc.recordTerminateSent(podKey)
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
			rowsAffected, err := pc.podRepo.UpdateByKeyAndStatusCounted(ctx, podKey, agentpod.StatusOrphaned, map[string]interface{}{
				"status":        agentpod.StatusRunning,
				"finished_at":   nil,
				"last_activity": now,
			})
			if err != nil {
				pc.logger.Error("failed to restore orphaned pod",
					"pod_key", podKey,
					"error", err)
			} else if rowsAffected > 0 {
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
	activePods, err := pc.podRepo.ListActiveByRunner(ctx, runnerID)
	if err != nil {
		pc.logger.Error("failed to get pods for reconciliation",
			"runner_id", runnerID,
			"error", err)
		return
	}

	// Mark pods that are in DB but not reported by runner.
	// Uses evidence-based orphaning: a pod must be missing for orphanMissThreshold
	// consecutive heartbeats before being orphaned, providing tolerance for
	// transient network issues and runner reconnections.
	for _, p := range activePods {
		if reportedPods[p.PodKey] {
			// Pod is present in heartbeat — clear any accumulated miss count
			pc.clearMissCount(p.PodKey)
			continue
		}

		missCount := pc.incrementMissCount(p.PodKey, runnerID)
		if missCount < orphanMissThreshold {
			pc.logger.Debug("pod not reported, waiting for more evidence",
				"pod_key", p.PodKey,
				"runner_id", runnerID,
				"miss_count", missCount,
				"threshold", orphanMissThreshold)
			continue
		}

		// Evidence threshold reached — orphan the pod
		pc.clearMissCount(p.PodKey)
		if err := pc.podRepo.MarkOrphaned(ctx, p, now); err != nil {
			pc.logger.Error("failed to mark pod as orphaned",
				"pod_key", p.PodKey,
				"error", err)
		} else {
			pc.logger.Warn("pod marked as orphaned (not reported by runner)",
				"pod_key", p.PodKey,
				"runner_id", runnerID,
				"miss_count", missCount)
			// Unregister from terminal router
			pc.terminalRouter.UnregisterPod(p.PodKey)

			// Notify status change for WebSocket event
			if pc.onStatusChange != nil {
				pc.onStatusChange(p.PodKey, agentpod.StatusOrphaned, "")
			}
		}
	}

	// Update current_pods count based on actual running pods reported by runner
	// This ensures consistency between runner state and database
	reportedKeys := make([]string, 0, len(reportedPods))
	for podKey := range reportedPods {
		reportedKeys = append(reportedKeys, podKey)
	}
	activePodCount, err := pc.podRepo.CountActiveByKeys(ctx, reportedKeys)
	if err != nil {
		pc.logger.Error("failed to count active pods for runner",
			"runner_id", runnerID,
			"error", err)
		return
	}

	// Update runner's current_pods to reflect actual active pods
	if err := pc.runnerRepo.SetPodCount(ctx, runnerID, activePodCount); err != nil {
		pc.logger.Error("failed to update runner current_pods",
			"runner_id", runnerID,
			"active_pods", activePodCount,
			"error", err)
	}
}
