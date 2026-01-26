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
