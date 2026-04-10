package grpc

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// GRPCCommandSender adapts GRPCRunnerAdapter to runner.RunnerCommandSender interface.
// This allows PodCoordinator to send commands via gRPC connections.
type GRPCCommandSender struct {
	adapter *GRPCRunnerAdapter
}

// NewGRPCCommandSender creates a new adapter.
func NewGRPCCommandSender(adapter *GRPCRunnerAdapter) *GRPCCommandSender {
	return &GRPCCommandSender{adapter: adapter}
}

// SendCreatePod sends a create pod command to a runner via gRPC.
// Uses Proto type directly - no conversion needed.
func (s *GRPCCommandSender) SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	slog.Info("sending create_pod command", "runner_id", runnerID, "pod_key", cmd.GetPodKey())
	if err := s.adapter.SendCreatePod(runnerID, cmd); err != nil {
		slog.Error("failed to send create_pod command", "runner_id", runnerID, "pod_key", cmd.GetPodKey(), "error", err)
		return err
	}
	return nil
}

// SendTerminatePod sends a terminate pod command to a runner via gRPC.
func (s *GRPCCommandSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	slog.Info("sending terminate_pod command", "runner_id", runnerID, "pod_key", podKey)
	if err := s.adapter.SendTerminatePod(runnerID, podKey, false); err != nil {
		slog.Error("failed to send terminate_pod command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendPodInput sends pod input to a runner via gRPC.
func (s *GRPCCommandSender) SendPodInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	if err := s.adapter.SendPodInput(runnerID, podKey, data); err != nil {
		slog.Error("failed to send pod_input command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendPrompt sends a prompt to a pod via gRPC.
func (s *GRPCCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	slog.Info("sending prompt command", "runner_id", runnerID, "pod_key", podKey)
	if err := s.adapter.SendPrompt(runnerID, podKey, prompt); err != nil {
		slog.Error("failed to send prompt command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendSubscribePod sends a subscribe pod command via gRPC.
func (s *GRPCCommandSender) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	slog.Info("sending subscribe_pod command", "runner_id", runnerID, "pod_key", podKey)
	if err := s.adapter.SendSubscribePod(runnerID, podKey, relayURL, runnerToken, includeSnapshot, snapshotHistory); err != nil {
		slog.Error("failed to send subscribe_pod command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendUnsubscribePod sends an unsubscribe pod command via gRPC.
func (s *GRPCCommandSender) SendUnsubscribePod(ctx context.Context, runnerID int64, podKey string) error {
	slog.Info("sending unsubscribe_pod command", "runner_id", runnerID, "pod_key", podKey)
	if err := s.adapter.SendUnsubscribePod(runnerID, podKey); err != nil {
		slog.Error("failed to send unsubscribe_pod command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendCreateAutopilot sends a create AutopilotController command to a runner via gRPC.
func (s *GRPCCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	slog.Info("sending create_autopilot command", "runner_id", runnerID, "autopilot_key", cmd.GetAutopilotKey())
	if err := s.adapter.SendCreateAutopilot(runnerID, cmd); err != nil {
		slog.Error("failed to send create_autopilot command", "runner_id", runnerID, "autopilot_key", cmd.GetAutopilotKey(), "error", err)
		return err
	}
	return nil
}

// SendAutopilotControl sends an AutopilotController control command to a runner via gRPC.
func (s *GRPCCommandSender) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	slog.Info("sending autopilot_control command", "runner_id", runnerID, "autopilot_key", cmd.GetAutopilotKey())
	if err := s.adapter.SendAutopilotControl(runnerID, cmd); err != nil {
		slog.Error("failed to send autopilot_control command", "runner_id", runnerID, "autopilot_key", cmd.GetAutopilotKey(), "error", err)
		return err
	}
	return nil
}

// SendUpdatePodPerpetual sends an update perpetual mode command to a runner via gRPC.
func (s *GRPCCommandSender) SendUpdatePodPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) error {
	slog.Info("sending update_pod_perpetual command", "runner_id", runnerID, "pod_key", podKey, "perpetual", perpetual)
	if err := s.adapter.SendUpdatePodPerpetual(runnerID, podKey, perpetual); err != nil {
		slog.Error("failed to send update_pod_perpetual command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// SendQuerySandboxes sends a sandbox query command to a runner via gRPC.
func (s *GRPCCommandSender) SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error {
	slog.Info("sending query_sandboxes command", "runner_id", runnerID, "request_id", requestID, "pod_count", len(podKeys))
	if err := s.adapter.SendQuerySandboxes(runnerID, requestID, podKeys); err != nil {
		slog.Error("failed to send query_sandboxes command", "runner_id", runnerID, "request_id", requestID, "error", err)
		return err
	}
	return nil
}

// SendObservePod sends an observe pod command to a runner via gRPC.
func (s *GRPCCommandSender) SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	slog.Info("sending observe_pod command", "runner_id", runnerID, "request_id", requestID, "pod_key", podKey)
	if err := s.adapter.SendObservePod(runnerID, requestID, podKey, lines, includeScreen); err != nil {
		slog.Error("failed to send observe_pod command", "runner_id", runnerID, "pod_key", podKey, "error", err)
		return err
	}
	return nil
}

// IsConnected checks if a runner is connected.
func (s *GRPCCommandSender) IsConnected(runnerID int64) bool {
	return s.adapter.IsConnected(runnerID)
}

// SendUpgradeRunner sends an upgrade command to a runner via gRPC.
func (s *GRPCCommandSender) SendUpgradeRunner(runnerID int64, requestID, targetVersion string, force bool) error {
	slog.Info("sending upgrade_runner command", "runner_id", runnerID, "request_id", requestID, "target_version", targetVersion, "force", force)
	if err := s.adapter.SendUpgradeRunner(runnerID, requestID, targetVersion, force); err != nil {
		slog.Error("failed to send upgrade_runner command", "runner_id", runnerID, "request_id", requestID, "error", err)
		return err
	}
	return nil
}

// SendUploadLogs sends a log upload command to a runner via gRPC.
func (s *GRPCCommandSender) SendUploadLogs(runnerID int64, requestID, presignedURL string, urlExpiresAt int64) error {
	slog.Info("sending upload_logs command", "runner_id", runnerID, "request_id", requestID)
	if err := s.adapter.SendUploadLogs(runnerID, requestID, presignedURL, urlExpiresAt); err != nil {
		slog.Error("failed to send upload_logs command", "runner_id", runnerID, "request_id", requestID, "error", err)
		return err
	}
	return nil
}

// Ensure GRPCCommandSender implements runner.RunnerCommandSender
var _ runner.RunnerCommandSender = (*GRPCCommandSender)(nil)

// Ensure GRPCCommandSender implements runner.SandboxQuerySender
var _ runner.SandboxQuerySender = (*GRPCCommandSender)(nil)

// Ensure GRPCCommandSender implements runner.UpgradeCommandSender
var _ runner.UpgradeCommandSender = (*GRPCCommandSender)(nil)

// Ensure GRPCCommandSender implements runner.LogUploadCommandSender
var _ runner.LogUploadCommandSender = (*GRPCCommandSender)(nil)
