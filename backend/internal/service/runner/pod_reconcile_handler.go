package runner

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

// reconcilePods syncs pod status between runner heartbeat and database
func (pc *PodCoordinator) reconcilePods(ctx context.Context, runnerID int64, reportedPods map[string]bool) {
	now := time.Now()

	// First, check reported pods against database status
	// - Restore orphaned pods that runner reports as active
	// - Recover initializing pods (PodCreated message loss / backend restart)
	// - Terminate pods that should not be running (terminated/completed in DB)
	for podKey := range reportedPods {
		pod, err := pc.podRepo.GetByKeyAndRunner(ctx, podKey, runnerID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				pc.logger.Warn("failed to lookup reported pod, will retry",
					"pod_key", podKey, "runner_id", runnerID, "error", err)
				continue
			}

			// Pod not in database — use evidence accumulation
			missCount := pc.incrementMissCount(podKey, runnerID)
			if missCount < orphanMissThreshold {
				pc.logger.Debug("unknown pod, waiting for more evidence",
					"pod_key", podKey, "miss_count", missCount, "threshold", orphanMissThreshold)
				continue
			}
			pc.clearMissCount(podKey)

			if pc.isTerminateCooldown(podKey) {
				continue
			}
			pc.recordTerminateSent(podKey)
			pc.logger.Warn("runner reported unknown pod, sending terminate",
				"pod_key", podKey, "runner_id", runnerID)
			if sendErr := pc.commandSender.SendTerminatePod(ctx, runnerID, podKey); sendErr != nil {
				pc.logger.Error("failed to send terminate for unknown pod",
					"pod_key", podKey, "error", sendErr)
			}
			continue
		}

		// Terminate pods that are already completed/terminated in DB
		if pod.Status == agentpod.StatusCompleted || pod.Status == agentpod.StatusTerminated {
			if pc.isTerminateCooldown(podKey) {
				continue
			}
			pc.recordTerminateSent(podKey)
			pc.logger.Warn("runner reported terminated pod, sending terminate",
				"pod_key", podKey, "runner_id", runnerID, "db_status", pod.Status)
			if sendErr := pc.commandSender.SendTerminatePod(ctx, runnerID, podKey); sendErr != nil {
				pc.logger.Error("failed to send terminate for completed pod",
					"pod_key", podKey, "error", sendErr)
			}
			continue
		}

		// Ensure pod is registered with terminal router (preserves existing VT state)
		pc.terminalRouter.EnsurePodRegistered(podKey, runnerID)

		// Restore orphaned pod
		if pod.Status == agentpod.StatusOrphaned {
			pc.recoverPodStatus(ctx, podKey, runnerID, agentpod.StatusOrphaned, now)
		}

		// Recover initializing pod that runner reports as active.
		// Uses evidence accumulation: only recover after the pod has been reported
		// in N consecutive heartbeats. This prevents premature recovery of pods
		// that are legitimately still initializing (slow git clone, large repo).
		// The counter naturally starts at 0 after backend restart, so it also
		// handles the "backend restarted, ackTracker lost" case correctly.
		//
		// Note: This is a best-effort recovery. Metadata set by handlePodCreated
		// (pty_pid, sandbox_path, branch_name) will be missing. Terminal routing
		// still works via EnsurePodRegistered above. Resume functionality (which
		// depends on sandbox_path) may not work for heartbeat-recovered pods.
		if pod.Status == agentpod.StatusInitializing {
			count := pc.incrementInitReportCount(podKey)
			if count >= initRecoverThreshold {
				pc.clearInitReportCount(podKey)
				pc.recoverPodStatus(ctx, podKey, runnerID, agentpod.StatusInitializing, now)
			}
		}
	}

	pc.reconcileMissingPods(ctx, runnerID, reportedPods, now)
	pc.syncPodCount(ctx, runnerID, reportedPods)
}

// recoverPodStatus transitions a pod from fromStatus to running.
func (pc *PodCoordinator) recoverPodStatus(ctx context.Context, podKey string, runnerID int64, fromStatus string, now time.Time) {
	updates := map[string]interface{}{
		"status":        agentpod.StatusRunning,
		"last_activity": now,
	}
	if fromStatus == agentpod.StatusOrphaned {
		updates["finished_at"] = nil
	}
	if fromStatus == agentpod.StatusInitializing {
		updates["started_at"] = now
	}

	rowsAffected, err := pc.podRepo.UpdateByKeyAndStatusCounted(ctx, podKey, fromStatus, updates)
	if err != nil {
		pc.logger.Error("failed to recover pod",
			"pod_key", podKey, "from_status", fromStatus, "error", err)
		return
	}
	if rowsAffected > 0 {
		if fromStatus == agentpod.StatusInitializing {
			pc.ackTracker.Resolve(podKey)
		}
		pc.logger.Info("recovered pod reported by runner heartbeat",
			"pod_key", podKey, "runner_id", runnerID, "from_status", fromStatus)
		if pc.onStatusChange != nil {
			pc.onStatusChange(podKey, agentpod.StatusRunning, "")
		}
	}
}

// reconcileMissingPods marks pods missing from heartbeat as orphaned.
func (pc *PodCoordinator) reconcileMissingPods(ctx context.Context, runnerID int64, reportedPods map[string]bool, now time.Time) {
	activePods, err := pc.podRepo.ListActiveByRunner(ctx, runnerID)
	if err != nil {
		pc.logger.Error("failed to get pods for reconciliation",
			"runner_id", runnerID, "error", err)
		return
	}

	for _, p := range activePods {
		if reportedPods[p.PodKey] {
			pc.clearMissCount(p.PodKey)
			continue
		}

		missCount := pc.incrementMissCount(p.PodKey, runnerID)
		if missCount < orphanMissThreshold {
			pc.logger.Debug("pod not reported, waiting for more evidence",
				"pod_key", p.PodKey, "runner_id", runnerID,
				"miss_count", missCount, "threshold", orphanMissThreshold)
			continue
		}

		pc.clearMissCount(p.PodKey)
		if err := pc.podRepo.MarkOrphaned(ctx, p, now); err != nil {
			pc.logger.Error("failed to mark pod as orphaned",
				"pod_key", p.PodKey, "error", err)
		} else {
			pc.logger.Warn("pod marked as orphaned (not reported by runner)",
				"pod_key", p.PodKey, "runner_id", runnerID, "miss_count", missCount)
			pc.terminalRouter.UnregisterPod(p.PodKey)
			if pc.onStatusChange != nil {
				pc.onStatusChange(p.PodKey, agentpod.StatusOrphaned, "")
			}
		}
	}
}

// syncPodCount updates the runner's pod count to match reality.
func (pc *PodCoordinator) syncPodCount(ctx context.Context, runnerID int64, reportedPods map[string]bool) {
	reportedKeys := make([]string, 0, len(reportedPods))
	for podKey := range reportedPods {
		reportedKeys = append(reportedKeys, podKey)
	}
	activePodCount, err := pc.podRepo.CountActiveByKeys(ctx, reportedKeys)
	if err != nil {
		pc.logger.Error("failed to count active pods for runner",
			"runner_id", runnerID, "error", err)
		return
	}
	if err := pc.runnerRepo.SetPodCount(ctx, runnerID, activePodCount); err != nil {
		pc.logger.Error("failed to update runner current_pods",
			"runner_id", runnerID, "active_pods", activePodCount, "error", err)
	}
}
