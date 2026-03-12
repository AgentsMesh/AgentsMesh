package runner

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== Callback Setters ====================

// SetHeartbeatCallback sets the heartbeat callback (Proto type)
func (cm *RunnerConnectionManager) SetHeartbeatCallback(fn func(runnerID int64, data *runnerv1.HeartbeatData)) {
	cm.onHeartbeat = fn
}

// SetPodCreatedCallback sets the pod created callback (Proto type)
func (cm *RunnerConnectionManager) SetPodCreatedCallback(fn func(runnerID int64, data *runnerv1.PodCreatedEvent)) {
	cm.onPodCreated = fn
}

// SetPodTerminatedCallback sets the pod terminated callback (Proto type)
func (cm *RunnerConnectionManager) SetPodTerminatedCallback(fn func(runnerID int64, data *runnerv1.PodTerminatedEvent)) {
	cm.onPodTerminated = fn
}

// SetPodErrorCallback sets the pod error callback (Proto type)
func (cm *RunnerConnectionManager) SetPodErrorCallback(fn func(runnerID int64, data *runnerv1.ErrorEvent)) {
	cm.onPodError = fn
}

// SetAgentStatusCallback sets the agent status callback (Proto type)
func (cm *RunnerConnectionManager) SetAgentStatusCallback(fn func(runnerID int64, data *runnerv1.AgentStatusEvent)) {
	cm.onAgentStatus = fn
}

// SetPtyResizedCallback sets the PTY resized callback (Proto type)
func (cm *RunnerConnectionManager) SetPtyResizedCallback(fn func(runnerID int64, data *runnerv1.PtyResizedEvent)) {
	cm.onPtyResized = fn
}

// SetPodInitProgressCallback sets the pod init progress callback (Proto type)
func (cm *RunnerConnectionManager) SetPodInitProgressCallback(fn func(runnerID int64, data *runnerv1.PodInitProgressEvent)) {
	cm.onPodInitProgress = fn
}

// SetRequestRelayTokenCallback sets the request relay token callback (Proto type)
func (cm *RunnerConnectionManager) SetRequestRelayTokenCallback(fn func(runnerID int64, data *runnerv1.RequestRelayTokenEvent)) {
	cm.onRequestRelayToken = fn
}

// SetDisconnectCallback sets the disconnect callback
func (cm *RunnerConnectionManager) SetDisconnectCallback(fn func(runnerID int64)) {
	cm.onDisconnect = fn
}

// SetInitializedCallback sets the initialized callback
func (cm *RunnerConnectionManager) SetInitializedCallback(fn func(runnerID int64, availableAgents []string)) {
	cm.onInitialized = fn
}

// SetInitFailedCallback sets the initialization failure callback
func (cm *RunnerConnectionManager) SetInitFailedCallback(fn func(runnerID int64, reason string)) {
	cm.onInitFailed = fn
}

// SetSandboxesStatusCallback sets the sandbox status callback (Proto type)
func (cm *RunnerConnectionManager) SetSandboxesStatusCallback(fn func(runnerID int64, data *runnerv1.SandboxesStatusEvent)) {
	cm.onSandboxesStatus = fn
}

// SetOSCNotificationCallback sets the OSC notification callback (Proto type)
// Called when terminal sends OSC 777 (iTerm2/Kitty) or OSC 9 (ConEmu/Windows Terminal) notifications
func (cm *RunnerConnectionManager) SetOSCNotificationCallback(fn func(runnerID int64, data *runnerv1.OSCNotificationEvent)) {
	cm.onOSCNotification = fn
}

// SetOSCTitleCallback sets the OSC title change callback (Proto type)
// Called when terminal sends OSC 0/2 window/tab title change sequences
func (cm *RunnerConnectionManager) SetOSCTitleCallback(fn func(runnerID int64, data *runnerv1.OSCTitleEvent)) {
	cm.onOSCTitle = fn
}

// SetInitTimeout sets the initialization timeout
func (cm *RunnerConnectionManager) SetInitTimeout(timeout time.Duration) {
	cm.initTimeout = timeout
}

// SetPingInterval sets the ping interval
func (cm *RunnerConnectionManager) SetPingInterval(interval time.Duration) {
	cm.pingInterval = interval
}

// SetAgentTypesProvider sets the agent types provider for initialization handshake
func (cm *RunnerConnectionManager) SetAgentTypesProvider(provider interfaces.AgentTypesProvider) {
	cm.agentTypesProvider = provider
}

// SetServerVersion sets the server version for initialization handshake
func (cm *RunnerConnectionManager) SetServerVersion(version string) {
	cm.serverVersion = version
}

// GetHeartbeatCallback returns the current heartbeat callback
func (cm *RunnerConnectionManager) GetHeartbeatCallback() func(runnerID int64, data *runnerv1.HeartbeatData) {
	return cm.onHeartbeat
}

// GetDisconnectCallback returns the current disconnect callback
func (cm *RunnerConnectionManager) GetDisconnectCallback() func(runnerID int64) {
	return cm.onDisconnect
}

// ==================== Token Usage Callback Setter ====================

// SetTokenUsageCallback sets the token usage report callback (Proto type)
func (cm *RunnerConnectionManager) SetTokenUsageCallback(fn func(runnerID int64, data *runnerv1.TokenUsageReport)) {
	cm.onTokenUsage = fn
}

// ==================== Upgrade Callback Setters ====================

// SetUpgradeStatusCallback sets the upgrade status callback (Proto type)
func (cm *RunnerConnectionManager) SetUpgradeStatusCallback(fn func(runnerID int64, data *runnerv1.UpgradeStatusEvent)) {
	cm.onUpgradeStatus = fn
}

// SetLogUploadStatusCallback sets the log upload status callback (Proto type)
func (cm *RunnerConnectionManager) SetLogUploadStatusCallback(fn func(runnerID int64, data *runnerv1.LogUploadStatusEvent)) {
	cm.onLogUploadStatus = fn
}

// ==================== AutopilotController Callback Setters ====================

// SetAutopilotStatusCallback sets the AutopilotController status callback (Proto type)
func (cm *RunnerConnectionManager) SetAutopilotStatusCallback(fn func(runnerID int64, data *runnerv1.AutopilotStatusEvent)) {
	cm.onAutopilotStatus = fn
}

// SetAutopilotIterationCallback sets the AutopilotController iteration callback (Proto type)
func (cm *RunnerConnectionManager) SetAutopilotIterationCallback(fn func(runnerID int64, data *runnerv1.AutopilotIterationEvent)) {
	cm.onAutopilotIteration = fn
}

// SetAutopilotCreatedCallback sets the AutopilotController created callback (Proto type)
func (cm *RunnerConnectionManager) SetAutopilotCreatedCallback(fn func(runnerID int64, data *runnerv1.AutopilotCreatedEvent)) {
	cm.onAutopilotCreated = fn
}

// SetAutopilotTerminatedCallback sets the AutopilotController terminated callback (Proto type)
func (cm *RunnerConnectionManager) SetAutopilotTerminatedCallback(fn func(runnerID int64, data *runnerv1.AutopilotTerminatedEvent)) {
	cm.onAutopilotTerminated = fn
}

// SetAutopilotThinkingCallback sets the AutopilotController thinking callback (Proto type)
// Called when Control Agent reports its decision-making process
func (cm *RunnerConnectionManager) SetAutopilotThinkingCallback(fn func(runnerID int64, data *runnerv1.AutopilotThinkingEvent)) {
	cm.onAutopilotThinking = fn
}
