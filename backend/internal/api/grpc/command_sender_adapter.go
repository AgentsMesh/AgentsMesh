package grpc

import (
	"context"

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
	return s.adapter.SendCreatePod(runnerID, cmd)
}

// SendTerminatePod sends a terminate pod command to a runner via gRPC.
func (s *GRPCCommandSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	return s.adapter.SendTerminatePod(runnerID, podKey, false)
}

// SendTerminalInput sends terminal input to a runner via gRPC.
func (s *GRPCCommandSender) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	return s.adapter.SendTerminalInput(runnerID, podKey, data)
}

// SendTerminalResize sends terminal resize to a runner via gRPC.
func (s *GRPCCommandSender) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	return s.adapter.SendTerminalResize(runnerID, podKey, int32(cols), int32(rows))
}

// SendTerminalRedraw triggers terminal redraw without changing size via gRPC.
func (s *GRPCCommandSender) SendTerminalRedraw(ctx context.Context, runnerID int64, podKey string) error {
	return s.adapter.SendTerminalRedraw(runnerID, podKey)
}

// SendPrompt sends a prompt to a pod via gRPC.
func (s *GRPCCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	return s.adapter.SendPrompt(runnerID, podKey, prompt)
}

// SendSubscribeTerminal sends a subscribe terminal command via gRPC.
func (s *GRPCCommandSender) SendSubscribeTerminal(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	return s.adapter.SendSubscribeTerminal(runnerID, podKey, relayURL, runnerToken, includeSnapshot, snapshotHistory)
}

// SendUnsubscribeTerminal sends an unsubscribe terminal command via gRPC.
func (s *GRPCCommandSender) SendUnsubscribeTerminal(ctx context.Context, runnerID int64, podKey string) error {
	return s.adapter.SendUnsubscribeTerminal(runnerID, podKey)
}

// SendCreateAutopilot sends a create AutopilotController command to a runner via gRPC.
func (s *GRPCCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	return s.adapter.SendCreateAutopilot(runnerID, cmd)
}

// SendAutopilotControl sends an AutopilotController control command to a runner via gRPC.
func (s *GRPCCommandSender) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	return s.adapter.SendAutopilotControl(runnerID, cmd)
}

// SendQuerySandboxes sends a sandbox query command to a runner via gRPC.
func (s *GRPCCommandSender) SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error {
	return s.adapter.SendQuerySandboxes(runnerID, requestID, podKeys)
}

// IsConnected checks if a runner is connected.
func (s *GRPCCommandSender) IsConnected(runnerID int64) bool {
	return s.adapter.IsConnected(runnerID)
}

// Ensure GRPCCommandSender implements runner.RunnerCommandSender
var _ runner.RunnerCommandSender = (*GRPCCommandSender)(nil)

// Ensure GRPCCommandSender implements runner.SandboxQuerySender
var _ runner.SandboxQuerySender = (*GRPCCommandSender)(nil)
