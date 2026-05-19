package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (pc *PodCoordinator) handleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	switch data.Status {
	case agentpod.AgentStatusExecuting, agentpod.AgentStatusWaiting, agentpod.AgentStatusIdle:
	default:
		pc.logger.Warn("invalid agent status received, ignoring",
			"runner_id", runnerID, "pod_key", data.PodKey, "status", data.Status)
		return
	}

	ctx := context.Background()

	updates := map[string]interface{}{
		"agent_status": data.Status,
	}

	if data.Status == agentpod.AgentStatusWaiting {
		now := time.Now()
		updates["agent_waiting_since"] = now
	} else {
		updates["agent_waiting_since"] = nil
	}

	if err := pc.podStore.UpdateAgentStatus(ctx, data.PodKey, updates); err != nil {
		pc.logger.Error("failed to update agent status",
			"pod_key", data.PodKey,
			"error", err)
		return
	}

	pc.logger.Debug("agent status changed",
		"pod_key", data.PodKey,
		"status", data.Status)

	if pc.onStatusChange != nil {
		pc.onStatusChange(data.PodKey, "", data.Status)
	}
}

func (pc *PodCoordinator) handlePodInitProgress(runnerID int64, data *runnerv1.PodInitProgressEvent) {
	if data.Phase == "received" {
		pc.ackTracker.Resolve(data.PodKey)
	}

	pc.logger.Debug("pod init progress",
		"pod_key", data.PodKey,
		"phase", data.Phase,
		"progress", data.Progress,
		"message", data.Message)

	if pc.onInitProgress != nil {
		pc.onInitProgress(data.PodKey, data.Phase, int(data.Progress), data.Message)
	}
}
