package runner

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (cm *RunnerConnectionManager) HandleHeartbeat(runnerID int64, data *runnerv1.HeartbeatData) {
	cm.UpdateHeartbeat(runnerID)
	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.SetLocalRelayURL(data.GetLocalRelayUrl())
	}
	if cm.onHeartbeat != nil {
		cm.onHeartbeat(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandlePodCreated(runnerID int64, data *runnerv1.PodCreatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("pod created event received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
	)
	if cm.onPodCreated != nil {
		cm.onPodCreated(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandlePodTerminated(runnerID int64, data *runnerv1.PodTerminatedEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("pod terminated event received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
		"exit_code", data.GetExitCode(),
	)
	if cm.onPodTerminated != nil {
		cm.onPodTerminated(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandlePodRestarting(runnerID int64, data *runnerv1.PodRestartingEvent) {
	cm.UpdateHeartbeat(runnerID)
	if cm.onPodRestarting != nil {
		cm.onPodRestarting(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandlePodError(runnerID int64, data *runnerv1.ErrorEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Error("pod error event received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
		"code", data.GetCode(),
		"message", data.GetMessage(),
	)
	if cm.onPodError != nil {
		cm.onPodError(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandleAgentStatus(runnerID int64, data *runnerv1.AgentStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("agent status event received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
		"status", data.GetStatus(),
	)
	if cm.onAgentStatus != nil {
		cm.onAgentStatus(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandlePodInitProgress(runnerID int64, data *runnerv1.PodInitProgressEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("pod init progress received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
		"phase", data.GetPhase(),
		"progress", data.GetProgress(),
	)
	if cm.onPodInitProgress != nil {
		cm.onPodInitProgress(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandleInitialized(runnerID int64, availableAgents []string) {
	cm.UpdateHeartbeat(runnerID)

	cm.logger.Info("runner initialized",
		"runner_id", runnerID,
		"available_agents", availableAgents,
	)

	if conn := cm.GetConnection(runnerID); conn != nil {
		conn.SetInitialized(true, availableAgents)
	}

	if cm.onInitialized != nil {
		cm.onInitialized(runnerID, availableAgents)
	}
}

func (cm *RunnerConnectionManager) HandleRequestRelayToken(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("relay token requested",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
	)
	if cm.onRequestRelayToken != nil {
		cm.onRequestRelayToken(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandleSandboxesStatus(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("sandboxes status received",
		"runner_id", runnerID,
		"request_id", data.GetRequestId(),
		"sandbox_count", len(data.GetSandboxes()),
	)
	if cm.onSandboxesStatus != nil {
		cm.onSandboxesStatus(runnerID, data)
	}
}

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

func (cm *RunnerConnectionManager) HandleObservePodResult(runnerID int64, data *runnerv1.ObservePodResult) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("observe pod result received",
		"runner_id", runnerID,
		"request_id", data.GetRequestId(),
		"pod_key", data.GetPodKey(),
	)
	if cm.onObservePodResult != nil {
		cm.onObservePodResult(runnerID, data)
	}
}

func (cm *RunnerConnectionManager) HandleTokenUsage(runnerID int64, data *runnerv1.TokenUsageReport) {
	cm.UpdateHeartbeat(runnerID)
	cm.logger.Info("token usage report received",
		"runner_id", runnerID,
		"pod_key", data.GetPodKey(),
	)
	if cm.onTokenUsage != nil {
		cm.onTokenUsage(runnerID, data)
	}
}
