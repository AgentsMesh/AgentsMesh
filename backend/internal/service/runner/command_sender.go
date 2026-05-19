package runner

import (
	"context"
	"log/slog"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type RunnerCommandSender interface {
	SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error

	SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error

	SendPodInput(ctx context.Context, runnerID int64, podKey string, data []byte) error

	SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error

	SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken, localToken string, includeSnapshot bool, snapshotHistory int32) error

	SendUnsubscribePod(ctx context.Context, runnerID int64, podKey string) error

	SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error

	SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error

	SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error

	SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error
}

type RunnerStateReader interface {
	GetRunnerLocalRelayURL(runnerID int64) string

	GetRunnerNodeID(runnerID int64) string
}

type NoOpCommandSender struct {
	logger *slog.Logger
}

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

func (n *NoOpCommandSender) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken, localToken string, includeSnapshot bool, snapshotHistory int32) error {
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

var _ RunnerCommandSender = (*NoOpCommandSender)(nil)

type SandboxQuerySender interface {
	SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error

	IsConnected(runnerID int64) bool
}

type UpgradeCommandSender interface {
	SendUpgradeRunner(runnerID int64, requestID, targetVersion string, force bool) error

	IsConnected(runnerID int64) bool
}

type LogUploadCommandSender interface {
	SendUploadLogs(runnerID int64, requestID, presignedURL string, urlExpiresAt int64) error

	IsConnected(runnerID int64) bool
}
