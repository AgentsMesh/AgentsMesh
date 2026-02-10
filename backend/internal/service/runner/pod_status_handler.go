package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// handleAgentStatus handles agent status change from runner (Proto type)
func (pc *PodCoordinator) handleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	// Validate agent status value
	switch data.Status {
	case agentpod.AgentStatusExecuting, agentpod.AgentStatusWaiting, agentpod.AgentStatusIdle:
		// valid
	default:
		pc.logger.Warn("invalid agent status received, ignoring",
			"runner_id", runnerID, "pod_key", data.PodKey, "status", data.Status)
		return
	}

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

// handlePodInitProgress handles pod init progress event from runner (Proto type)
func (pc *PodCoordinator) handlePodInitProgress(runnerID int64, data *runnerv1.PodInitProgressEvent) {
	pc.logger.Debug("pod init progress",
		"pod_key", data.PodKey,
		"phase", data.Phase,
		"progress", data.Progress,
		"message", data.Message)

	// Notify via callback (to publish realtime event)
	if pc.onInitProgress != nil {
		pc.onInitProgress(data.PodKey, data.Phase, int(data.Progress), data.Message)
	}
}
