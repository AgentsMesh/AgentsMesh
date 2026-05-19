package runner

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func (cm *RunnerConnectionManager) SetHeartbeatCallback(fn func(runnerID int64, data *runnerv1.HeartbeatData)) {
	cm.onHeartbeat = fn
}

func (cm *RunnerConnectionManager) SetPodCreatedCallback(fn func(runnerID int64, data *runnerv1.PodCreatedEvent)) {
	cm.onPodCreated = fn
}

func (cm *RunnerConnectionManager) SetPodTerminatedCallback(fn func(runnerID int64, data *runnerv1.PodTerminatedEvent)) {
	cm.onPodTerminated = fn
}

func (cm *RunnerConnectionManager) SetPodErrorCallback(fn func(runnerID int64, data *runnerv1.ErrorEvent)) {
	cm.onPodError = fn
}

func (cm *RunnerConnectionManager) SetAgentStatusCallback(fn func(runnerID int64, data *runnerv1.AgentStatusEvent)) {
	cm.onAgentStatus = fn
}

func (cm *RunnerConnectionManager) SetPodInitProgressCallback(fn func(runnerID int64, data *runnerv1.PodInitProgressEvent)) {
	cm.onPodInitProgress = fn
}

func (cm *RunnerConnectionManager) SetRequestRelayTokenCallback(fn func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)) {
	cm.onRequestRelayToken = fn
}

func (cm *RunnerConnectionManager) SetDisconnectCallback(fn func(runnerID int64)) {
	cm.onDisconnect = fn
}

func (cm *RunnerConnectionManager) SetInitializedCallback(fn func(runnerID int64, availableAgents []string)) {
	cm.onInitialized = fn
}

func (cm *RunnerConnectionManager) SetInitFailedCallback(fn func(runnerID int64, reason string)) {
	cm.onInitFailed = fn
}

func (cm *RunnerConnectionManager) SetSandboxesStatusCallback(fn func(runnerID int64, data *runnerv1.SandboxesStatusEvent)) {
	cm.onSandboxesStatus = fn
}

func (cm *RunnerConnectionManager) SetOSCNotificationCallback(fn func(runnerID int64, data *runnerv1.OSCNotificationEvent)) {
	cm.onOSCNotification = fn
}

func (cm *RunnerConnectionManager) SetOSCTitleCallback(fn func(runnerID int64, data *runnerv1.OSCTitleEvent)) {
	cm.onOSCTitle = fn
}

func (cm *RunnerConnectionManager) SetInitTimeout(timeout time.Duration) {
	cm.initTimeout = timeout
}

func (cm *RunnerConnectionManager) SetPingInterval(interval time.Duration) {
	cm.pingInterval = interval
}

func (cm *RunnerConnectionManager) SetAgentsProvider(provider interfaces.AgentsProvider) {
	cm.agentsProvider = provider
}

func (cm *RunnerConnectionManager) SetServerVersion(version string) {
	cm.serverVersion = version
}

func (cm *RunnerConnectionManager) GetHeartbeatCallback() func(runnerID int64, data *runnerv1.HeartbeatData) {
	return cm.onHeartbeat
}

func (cm *RunnerConnectionManager) GetDisconnectCallback() func(runnerID int64) {
	return cm.onDisconnect
}

func (cm *RunnerConnectionManager) SetObservePodResultCallback(fn func(runnerID int64, data *runnerv1.ObservePodResult)) {
	cm.onObservePodResult = fn
}

func (cm *RunnerConnectionManager) SetTokenUsageCallback(fn func(runnerID int64, data *runnerv1.TokenUsageReport)) {
	cm.onTokenUsage = fn
}

func (cm *RunnerConnectionManager) SetUpgradeStatusCallback(fn func(runnerID int64, data *runnerv1.UpgradeStatusEvent)) {
	cm.onUpgradeStatus = fn
}

func (cm *RunnerConnectionManager) SetLogUploadStatusCallback(fn func(runnerID int64, data *runnerv1.LogUploadStatusEvent)) {
	cm.onLogUploadStatus = fn
}

func (cm *RunnerConnectionManager) SetPodRestartingCallback(fn func(runnerID int64, data *runnerv1.PodRestartingEvent)) {
	cm.onPodRestarting = fn
}

func (cm *RunnerConnectionManager) SetAutopilotStatusCallback(fn func(runnerID int64, data *runnerv1.AutopilotStatusEvent)) {
	cm.onAutopilotStatus = fn
}

func (cm *RunnerConnectionManager) SetAutopilotIterationCallback(fn func(runnerID int64, data *runnerv1.AutopilotIterationEvent)) {
	cm.onAutopilotIteration = fn
}

func (cm *RunnerConnectionManager) SetAutopilotCreatedCallback(fn func(runnerID int64, data *runnerv1.AutopilotCreatedEvent)) {
	cm.onAutopilotCreated = fn
}

func (cm *RunnerConnectionManager) SetAutopilotTerminatedCallback(fn func(runnerID int64, data *runnerv1.AutopilotTerminatedEvent)) {
	cm.onAutopilotTerminated = fn
}

func (cm *RunnerConnectionManager) SetAutopilotThinkingCallback(fn func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)) {
	cm.onAutopilotThinking = fn
}
