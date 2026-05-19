package runner

import (
	"context"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const (
	perpetualCircuitBreakerWindow    = 5 * time.Minute
	perpetualCircuitBreakerThreshold = 3
)

func (pc *PodCoordinator) handlePodRestarting(runnerID int64, data *runnerv1.PodRestartingEvent) {
	ctx := context.Background()
	now := time.Now()

	var prevLastRestartAt *time.Time
	if data.RestartCount >= perpetualCircuitBreakerThreshold {
		pod, err := pc.podStore.GetByKey(ctx, data.PodKey)
		if err == nil && pod != nil {
			prevLastRestartAt = pod.LastRestartAt
		}
	}

	updates := map[string]interface{}{
		"restart_count":   int(data.RestartCount),
		"last_restart_at": now,
	}
	if data.NewPid > 0 {
		updates["pty_pid"] = int(data.NewPid)
	}
	if _, err := pc.podStore.UpdateByKey(ctx, data.PodKey, updates); err != nil {
		pc.logger.Error("failed to update pod restart info",
			"pod_key", data.PodKey, "error", err)
		return
	}

	pc.logger.Info("perpetual pod restarting",
		"pod_key", data.PodKey,
		"runner_id", runnerID,
		"exit_code", data.ExitCode,
		"restart_count", data.RestartCount)

	if pc.onPodRestarting != nil {
		pc.onPodRestarting(data.PodKey, data.ExitCode, data.RestartCount)
	}

	if data.RestartCount >= perpetualCircuitBreakerThreshold && prevLastRestartAt != nil {
		if now.Sub(*prevLastRestartAt) < perpetualCircuitBreakerWindow {
			pc.logger.Warn("perpetual pod circuit breaker triggered",
				"pod_key", data.PodKey, "restart_count", data.RestartCount,
				"window", now.Sub(*prevLastRestartAt))
			if err := pc.commandSender.SendTerminatePod(ctx, runnerID, data.PodKey); err != nil {
				pc.logger.Error("failed to send terminate for circuit breaker",
					"pod_key", data.PodKey, "error", err)
			}
		}
	}
}
