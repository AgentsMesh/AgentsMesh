package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// handleRunnerDisconnect handles runner disconnection
func (pc *PodCoordinator) handleRunnerDisconnect(runnerID int64) {
	ctx := context.Background()

	// Mark runner as offline, but don't immediately orphan running pods
	// (they will be orphaned by reconcilePods if runner doesn't reconnect)
	if err := pc.runnerRepo.UpdateFields(ctx, runnerID, map[string]interface{}{
		"status": "offline",
	}); err != nil {
		pc.logger.Error("failed to mark runner as offline",
			"runner_id", runnerID,
			"error", err)
	}

	// Clear relay connection cache for this runner
	pc.relayConnectionCache.Delete(runnerID)

	// Clear miss counters for this runner's pods to prevent stale counts
	// from affecting reconciliation after reconnection.
	pc.clearMissCountsForRunner(runnerID)

	// Fail initializing pods immediately — these pods never started running,
	// so there's no point waiting for the runner to reconnect. Unlike running
	// pods (which may survive a brief network glitch), initializing pods have
	// no terminal session to preserve.
	pc.failInitializingPodsForRunner(ctx, runnerID)

	pc.logger.Info("runner disconnected, running pods will be reconciled on reconnect",
		"runner_id", runnerID)
}

// failInitializingPodsForRunner marks all initializing pods for the given runner as error.
func (pc *PodCoordinator) failInitializingPodsForRunner(ctx context.Context, runnerID int64) {
	pods, err := pc.podRepo.ListInitializingByRunner(ctx, runnerID)
	if err != nil {
		pc.logger.Error("failed to list initializing pods for disconnected runner",
			"runner_id", runnerID, "error", err)
		return
	}

	now := time.Now()
	for _, pod := range pods {
		rowsAffected, err := pc.podRepo.UpdateByKeyAndStatusCounted(ctx, pod.PodKey, agentpod.StatusInitializing, map[string]interface{}{
			"status":        agentpod.StatusError,
			"error_code":    ErrCodeRunnerDisconnected,
			"error_message": "Runner disconnected during pod initialization.",
			"finished_at":   now,
		})
		if err != nil {
			pc.logger.Error("failed to fail initializing pod on disconnect",
				"pod_key", pod.PodKey, "error", err)
			continue
		}
		if rowsAffected > 0 {
			_ = pc.runnerRepo.DecrementPods(ctx, runnerID)
			pc.ackTracker.Remove(pod.PodKey) // Cancel any pending ACK wait
			if pc.onStatusChange != nil {
				pc.onStatusChange(pod.PodKey, agentpod.StatusError, "")
			}
			pc.logger.Warn("initializing pod failed due to runner disconnect",
				"pod_key", pod.PodKey, "runner_id", runnerID)
		}
	}
}
