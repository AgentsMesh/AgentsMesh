package runner

import (
	"context"
	"log/slog"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// RunnerCommandSender defines the interface for sending commands to runners.
// This interface allows PodCoordinator and PodRouter to work with different implementations:
// - GRPCCommandSender (gRPC adapter in api/grpc package)
//
// Note: RunnerConnectionManager does NOT implement this interface.
// It only manages connection state; command sending goes through GRPCCommandSender.
// To check connection status, use RunnerConnectionManager.IsConnected directly.
type RunnerCommandSender interface {
	// SendCreatePod sends a create pod command to a runner.
	// Uses Proto type directly for zero-copy message passing.
	SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error

	// SendTerminatePod sends a terminate pod command to a runner.
	SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error

	// SendPodInput sends pod input to a runner.
	SendPodInput(ctx context.Context, runnerID int64, podKey string, data []byte) error

	// SendPrompt sends a prompt to a pod.
	SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error

	// SendSubscribePod sends a subscribe pod command to a runner.
	// relayURL is the public URL via reverse proxy (e.g. wss://example.com/relay).
	SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error

	// SendUnsubscribePod sends an unsubscribe pod command to a runner.
	// This notifies the runner to disconnect from Relay when all browsers have disconnected.
	SendUnsubscribePod(ctx context.Context, runnerID int64, podKey string) error

	// SendObservePod sends an observe pod command to a runner.
	// Response is delivered via callback registered in RunnerConnectionManager.
	SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error

	// SendCreateAutopilot sends a create AutopilotController command to a runner.
	SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error

	// SendAutopilotControl sends an AutopilotController control command to a runner.
	SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error

	// SendUpdatePodPerpetual notifies a runner to update the perpetual flag for a running pod.
	SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error
}

// NoOpCommandSender is a fallback implementation that logs warnings.
// Used when gRPC/mTLS is not configured.
type NoOpCommandSender struct {
	logger *slog.Logger
}

// NewNoOpCommandSender creates a new no-op command sender.
func NewNoOpCommandSender(logger *slog.Logger) *NoOpCommandSender {
	return &NoOpCommandSender{logger: logger}
}

func (n *NoOpCommandSender) SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	n.logger.Warn("command sender not configured, cannot create pod",
		"runner_id", runnerID, "pod_key", cmd.PodKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	n.logger.Warn("command sender not configured, cannot terminate pod",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendPodInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	n.logger.Warn("command sender not configured, cannot send pod input",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	n.logger.Warn("command sender not configured, cannot send prompt",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	n.logger.Warn("command sender not configured, cannot send subscribe pod",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendUnsubscribePod(ctx context.Context, runnerID int64, podKey string) error {
	n.logger.Warn("command sender not configured, cannot send unsubscribe pod",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	n.logger.Warn("command sender not configured, cannot send observe pod",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	n.logger.Warn("command sender not configured, cannot create autopilot",
		"runner_id", runnerID, "autopilot_key", cmd.AutopilotKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	n.logger.Warn("command sender not configured, cannot send autopilot control",
		"runner_id", runnerID, "autopilot_key", cmd.AutopilotKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error {
	n.logger.Warn("command sender not configured, cannot update pod perpetual",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

// Ensure NoOpCommandSender implements RunnerCommandSender
var _ RunnerCommandSender = (*NoOpCommandSender)(nil)

// SandboxQuerySender defines the interface for sending sandbox queries to runners.
// This is a separate interface from RunnerCommandSender (Interface Segregation).
type SandboxQuerySender interface {
	// SendQuerySandboxes sends a query sandboxes command to a runner.
	// Response is delivered via callback registered in RunnerConnectionManager.
	SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error

	// IsConnected checks if a runner is connected.
	IsConnected(runnerID int64) bool
}

// UpgradeCommandSender defines the interface for sending upgrade commands to runners.
// This is a separate interface from RunnerCommandSender (Interface Segregation).
type UpgradeCommandSender interface {
	// SendUpgradeRunner sends an upgrade command to a runner.
	SendUpgradeRunner(runnerID int64, requestID, targetVersion string, force bool) error

	// IsConnected checks if a runner is connected.
	IsConnected(runnerID int64) bool
}

// LogUploadCommandSender defines the interface for sending log upload commands to runners.
// This is a separate interface from RunnerCommandSender (Interface Segregation).
type LogUploadCommandSender interface {
	// SendUploadLogs sends a log upload command to a runner.
	SendUploadLogs(runnerID int64, requestID, presignedURL string, urlExpiresAt int64) error

	// IsConnected checks if a runner is connected.
	IsConnected(runnerID int64) bool
}
