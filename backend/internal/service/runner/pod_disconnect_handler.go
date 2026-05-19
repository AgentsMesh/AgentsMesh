package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func (pc *PodCoordinator) handleRunnerDisconnect(runnerID int64) {
	ctx := context.Background()

	if err := pc.runnerRepo.UpdateFields(ctx, runnerID, map[string]interface{}{
		"status": "offline",
	}); err != nil {
		pc.logger.Error("failed to mark runner as offline",
			"runner_id", runnerID,
			"error", err)
	}

	pc.relayConnectionCache.Delete(runnerID)

	pc.clearMissCountsForRunner(runnerID)

	pc.failInitializingPodsForRunner(ctx, runnerID)

	pc.logger.Info("runner disconnected, running pods will be reconciled on reconnect",
		"runner_id", runnerID)
}

func (pc *PodCoordinator) failInitializingPodsForRunner(ctx context.Context, runnerID int64) {
	pods, err := pc.podStore.ListInitializingByRunner(ctx, runnerID)
	if err != nil {
		pc.logger.Error("failed to list initializing pods for disconnected runner",
			"runner_id", runnerID, "error", err)
		return
	}

	now := time.Now()
	for _, pod := range pods {
		rowsAffected, err := pc.podStore.UpdateByKeyAndStatusCounted(ctx, pod.PodKey, agentpod.StatusInitializing, map[string]interface{}{
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
