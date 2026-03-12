package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Proto Message Handlers (called by GRPCRunnerAdapter) ====================

// HandleHeartbeat handles heartbeat from a runner (Proto type)
func (cm *RunnerConnectionManager) HandleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onHeartbeat != nil {
		cm.onHeartbeat(runnerID, data)
	}
}

// HandlePodCreated handles pod created event (Proto type)
func (cm *RunnerConnectionManager) HandlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodCreated != nil {
		cm.onPodCreated(runnerID, data)
	}
}

// HandlePodTerminated handles pod terminated event (Proto type)
func (cm *RunnerConnectionManager) HandlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodTerminated != nil {
		cm.onPodTerminated(runnerID, data)
	}
}

// HandlePodError handles pod error event (Proto type)
func (cm *RunnerConnectionManager) HandlePodError(runnerID int64, data *runnerv1.ErrorEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodError != nil {
		cm.onPodError(runnerID, data)
	}
}

// NOTE: HandleTerminalOutput removed - terminal output is exclusively streamed via Relay

// HandleAgentStatus handles agent status event (Proto type)
func (cm *RunnerConnectionManager) HandleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onAgentStatus != nil {
		cm.onAgentStatus(runnerID, data)
	}
}

// HandlePtyResized handles PTY resized event (Proto type)
func (cm *RunnerConnectionManager) HandlePtyResized(runnerID int64, data *runnerv1.PtyResizedEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPtyResized != nil {
		cm.onPtyResized(runnerID, data)
	}
}

// HandlePodInitProgress handles pod init progress event (Proto type)
func (cm *RunnerConnectionManager) HandlePodInitProgress(runnerID int64, data *runnerv1.PodInitProgressEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodInitProgress != nil {
		cm.onPodInitProgress(runnerID, data)
	}
}

// HandleInitialized handles initialized confirmation (Proto type)
func (cm *RunnerConnectionManager) HandleInitialized(runnerID int64, availableAgents []string) {
	cm.UpdateHeartbeat(runnerID)

	// Mark connection as initialized
	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.SetInitialized(true, availableAgents)
	}

	if cm.onInitialized != nil {
		cm.onInitialized(runnerID, availableAgents)
	}
}

// HandleRequestRelayToken handles relay token refresh request (Proto type)
func (cm *RunnerConnectionManager) HandleRequestRelayToken(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onRequestRelayToken != nil {
		cm.onRequestRelayToken(runnerID, data)
	}
}

// HandleSandboxesStatus handles sandbox status response event (Proto type)
func (cm *RunnerConnectionManager) HandleSandboxesStatus(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onSandboxesStatus != nil {
		cm.onSandboxesStatus(runnerID, data)
	}
}

// HandleOSCNotification handles OSC notification event from terminal (Proto type)
// OSC 777 (iTerm2/Kitty) or OSC 9 (ConEmu/Windows Terminal) desktop notification
func (cm *RunnerConnectionManager) HandleOSCNotification(runnerID int64, data *runnerv1.OSCNotificationEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Debug("received OSC notification",
		"runner_id", runnerID,
		"pod_key", data.PodKey,
		"title", data.Title,
		"body", data.Body,
	)
	if cm.onOSCNotification != nil {
		cm.onOSCNotification(runnerID, data)
	}
}

// HandleOSCTitle handles OSC title change event from terminal (Proto type)
// OSC 0/2 window/tab title change
func (cm *RunnerConnectionManager) HandleOSCTitle(runnerID int64, data *runnerv1.OSCTitleEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Debug("received OSC title change",
		"runner_id", runnerID,
		"pod_key", data.PodKey,
		"title", data.Title,
	)
	if cm.onOSCTitle != nil {
		cm.onOSCTitle(runnerID, data)
	}
}

// ==================== Token Usage Handler ====================

// HandleTokenUsage handles token usage report from runner (Proto type)
func (cm *RunnerConnectionManager) HandleTokenUsage(runnerID int64, data *runnerv1.TokenUsageReport) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onTokenUsage != nil {
		cm.onTokenUsage(runnerID, data)
	}
}

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
