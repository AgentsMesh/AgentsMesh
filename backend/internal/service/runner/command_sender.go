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

// Ensure NoOpCommandSender implements RunnerCommandSender
var _ RunnerCommandSender = (*NoOpCommandSender)(nil)
