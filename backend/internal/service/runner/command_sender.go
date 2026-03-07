package runner

import (
	"context"
	"log/slog"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// RunnerCommandSender defines the interface for sending commands to runners.
// This interface allows PodCoordinator and TerminalRouter to work with different implementations:
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

	// SendTerminalInput sends terminal input to a runner.
	SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error

	// SendTerminalResize sends terminal resize to a runner.
	SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error

	// SendTerminalRedraw triggers a terminal redraw without changing size.
	// Used to restore terminal state after server restart.
	SendTerminalRedraw(ctx context.Context, runnerID int64, podKey string) error

	// SendPrompt sends a prompt to a pod.
	SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error

	// SendSubscribeTerminal sends a subscribe terminal command to a runner.
	// relayURL is the public URL via reverse proxy (e.g. wss://example.com/relay).
	SendSubscribeTerminal(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error

	// SendUnsubscribeTerminal sends an unsubscribe terminal command to a runner.
	// This notifies the runner that all browsers have disconnected and it should disconnect from Relay.
	SendUnsubscribeTerminal(ctx context.Context, runnerID int64, podKey string) error

	// SendCreateAutopilot sends a create AutopilotController command to a runner.
	SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error

	// SendAutopilotControl sends an AutopilotController control command to a runner.
	SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error
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

func (n *NoOpCommandSender) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	n.logger.Warn("command sender not configured, cannot send terminal input",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	n.logger.Warn("command sender not configured, cannot send terminal resize",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendTerminalRedraw(ctx context.Context, runnerID int64, podKey string) error {
	n.logger.Warn("command sender not configured, cannot send terminal redraw",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	n.logger.Warn("command sender not configured, cannot send prompt",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendSubscribeTerminal(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	n.logger.Warn("command sender not configured, cannot send subscribe terminal",
		"runner_id", runnerID, "pod_key", podKey)
	return ErrCommandSenderNotSet
}

func (n *NoOpCommandSender) SendUnsubscribeTerminal(ctx context.Context, runnerID int64, podKey string) error {
	n.logger.Warn("command sender not configured, cannot send unsubscribe terminal",
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
