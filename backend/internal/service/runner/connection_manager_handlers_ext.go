package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Upgrade Event Handlers ====================

// HandleUpgradeStatus handles upgrade status event from runner (Proto type)
func (cm *RunnerConnectionManager) HandleUpgradeStatus(runnerID int64, data *runnerv1.UpgradeStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("received upgrade status",
		"runner_id", runnerID,
		"request_id", data.RequestId,
		"phase", data.Phase,
		"progress", data.Progress,
		"message", data.Message,
		"error", data.Error,
	)
	if cm.onUpgradeStatus != nil {
		cm.onUpgradeStatus(runnerID, data)
	}
}

// HandleLogUploadStatus handles log upload status event from runner (Proto type)
func (cm *RunnerConnectionManager) HandleLogUploadStatus(runnerID int64, data *runnerv1.LogUploadStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("received log upload status",
		"runner_id", runnerID,
		"request_id", data.RequestId,
		"phase", data.Phase,
		"progress", data.Progress,
		"message", data.Message,
		"error", data.Error,
		"size_bytes", data.SizeBytes,
	)
	if cm.onLogUploadStatus != nil {
		cm.onLogUploadStatus(runnerID, data)
	}
}

// ==================== AutopilotController Event Handlers ====================

// HandleAutopilotStatus handles AutopilotController status update event (Proto type)
func (cm *RunnerConnectionManager) HandleAutopilotStatus(runnerID int64, data *runnerv1.AutopilotStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Debug("received AutopilotController status",
		"runner_id", runnerID,
		"autopilot_key", data.AutopilotKey,
		"phase", data.Status.GetPhase(),
	)
	if cm.onAutopilotStatus != nil {
		cm.onAutopilotStatus(runnerID, data)
	}
}

// HandleAutopilotIteration handles AutopilotController iteration event (Proto type)
func (cm *RunnerConnectionManager) HandleAutopilotIteration(runnerID int64, data *runnerv1.AutopilotIterationEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Debug("received AutopilotController iteration",
		"runner_id", runnerID,
		"autopilot_key", data.AutopilotKey,
		"iteration", data.Iteration,
		"phase", data.Phase,
	)
	if cm.onAutopilotIteration != nil {
		cm.onAutopilotIteration(runnerID, data)
	}
}

// HandleAutopilotCreated handles AutopilotController created event (Proto type)
func (cm *RunnerConnectionManager) HandleAutopilotCreated(runnerID int64, data *runnerv1.AutopilotCreatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("AutopilotController created",
		"runner_id", runnerID,
		"autopilot_key", data.AutopilotKey,
		"pod_key", data.PodKey,
	)
	if cm.onAutopilotCreated != nil {
		cm.onAutopilotCreated(runnerID, data)
	}
}

// HandleAutopilotTerminated handles AutopilotController terminated event (Proto type)
func (cm *RunnerConnectionManager) HandleAutopilotTerminated(runnerID int64, data *runnerv1.AutopilotTerminatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("AutopilotController terminated",
		"runner_id", runnerID,
		"autopilot_key", data.AutopilotKey,
		"reason", data.Reason,
	)
	if cm.onAutopilotTerminated != nil {
		cm.onAutopilotTerminated(runnerID, data)
	}
}

// HandleAutopilotThinking handles AutopilotController thinking event (Proto type)
// This event exposes the Control Agent's decision-making process to the user
func (cm *RunnerConnectionManager) HandleAutopilotThinking(runnerID int64, data *runnerv1.AutopilotThinkingEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Debug("received AutopilotController thinking",
		"runner_id", runnerID,
		"autopilot_key", data.AutopilotKey,
		"iteration", data.Iteration,
		"decision_type", data.DecisionType,
	)
	if cm.onAutopilotThinking != nil {
		cm.onAutopilotThinking(runnerID, data)
	}
}
