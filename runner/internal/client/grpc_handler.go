// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"io"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// readLoop reads messages from the gRPC stream.
// The done channel is closed when the loop exits to notify other goroutines.
func (c *GRPCConnection) readLoop(ctx context.Context, done chan<- struct{}) {
	defer close(done) // Signal exit to other goroutines
	log := logger.GRPC()
	log.Info("Read loop starting")
	for {
		msg, err := c.stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Info("Stream ended (EOF)")
				return
			}
			if status.Code(err) == codes.Canceled {
				logger.GRPCTrace().Trace("Stream cancelled")
			} else {
				log.Error("Stream error", "error", err)
			}
			return
		}
		c.handleServerMessage(msg)
	}
}

// handleServerMessage dispatches received server messages to appropriate handlers.
func (c *GRPCConnection) handleServerMessage(msg *runnerv1.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.ServerMessage_InitializeResult:
		c.handleInitializeResult(payload.InitializeResult)

	case *runnerv1.ServerMessage_CreatePod:
		c.handleCreatePod(payload.CreatePod)

	case *runnerv1.ServerMessage_TerminatePod:
		c.handleTerminatePod(payload.TerminatePod)

	case *runnerv1.ServerMessage_TerminalInput:
		c.handleTerminalInput(payload.TerminalInput)

	case *runnerv1.ServerMessage_TerminalResize:
		c.handleTerminalResize(payload.TerminalResize)

	case *runnerv1.ServerMessage_TerminalRedraw:
		c.handleTerminalRedraw(payload.TerminalRedraw)

	case *runnerv1.ServerMessage_SendPrompt:
		c.handleSendPrompt(payload.SendPrompt)

	case *runnerv1.ServerMessage_SubscribeTerminal:
		c.handleSubscribeTerminal(payload.SubscribeTerminal)

	case *runnerv1.ServerMessage_UnsubscribeTerminal:
		c.handleUnsubscribeTerminal(payload.UnsubscribeTerminal)

	case *runnerv1.ServerMessage_QuerySandboxes:
		c.handleQuerySandboxes(payload.QuerySandboxes)

	case *runnerv1.ServerMessage_CreateAutopilot:
		c.handleCreateAutopilot(payload.CreateAutopilot)

	case *runnerv1.ServerMessage_AutopilotControl:
		c.handleAutopilotControl(payload.AutopilotControl)

	default:
		logger.GRPC().Warn("Unknown server message type")
	}
}

// handleInitializeResult handles initialize_result from server.
func (c *GRPCConnection) handleInitializeResult(result *runnerv1.InitializeResult) {
	logger.GRPC().Debug("Received initialize_result", "version", result.ServerInfo.Version)
	// Convert to internal type and send to channel
	select {
	case c.initResultCh <- result:
	default:
		logger.GRPC().Warn("Initialize result channel full, dropping")
	}
}

// handleCreatePod handles create_pod command from server.
// Passes Proto type directly to handler for zero-copy message passing.
func (c *GRPCConnection) handleCreatePod(cmd *runnerv1.CreatePodCommand) {
	log := logger.GRPC()
	log.Info("Received create_pod", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring create_pod")
		return
	}

	// Pass Proto type directly - no conversion needed
	if err := c.handler.OnCreatePod(cmd); err != nil {
		log.Error("Failed to create pod", "pod_key", cmd.PodKey, "error", err)
		c.sendError(cmd.PodKey, "create_pod_failed", err.Error())
	}
}

// handleTerminatePod handles terminate_pod command from server.
func (c *GRPCConnection) handleTerminatePod(cmd *runnerv1.TerminatePodCommand) {
	log := logger.GRPC()
	log.Info("Received terminate_pod", "pod_key", cmd.PodKey, "force", cmd.Force)
	if c.handler == nil {
		log.Warn("No handler set, ignoring terminate_pod")
		return
	}

	req := TerminatePodRequest{PodKey: cmd.PodKey}
	if err := c.handler.OnTerminatePod(req); err != nil {
		log.Error("Failed to terminate pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalInput handles terminal_input command from server.
func (c *GRPCConnection) handleTerminalInput(cmd *runnerv1.TerminalInputCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalInputRequest{
		PodKey: cmd.PodKey,
		Data:   cmd.Data, // gRPC uses native bytes, no encoding needed
	}
	if err := c.handler.OnTerminalInput(req); err != nil {
		logger.GRPC().Error("Failed to send terminal input to pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalResize handles terminal_resize command from server.
func (c *GRPCConnection) handleTerminalResize(cmd *runnerv1.TerminalResizeCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalResizeRequest{
		PodKey: cmd.PodKey,
		Cols:   uint16(cmd.Cols),
		Rows:   uint16(cmd.Rows),
	}
	if err := c.handler.OnTerminalResize(req); err != nil {
		logger.GRPC().Error("Failed to resize terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalRedraw handles terminal_redraw command from server.
// Uses resize +1/-1 trick to trigger terminal redraw for state recovery.
func (c *GRPCConnection) handleTerminalRedraw(cmd *runnerv1.TerminalRedrawCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalRedrawRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnTerminalRedraw(req); err != nil {
		logger.GRPC().Error("Failed to redraw terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSendPrompt handles send_prompt command from server.
func (c *GRPCConnection) handleSendPrompt(cmd *runnerv1.SendPromptCommand) {
	logger.GRPC().Debug("Received send_prompt", "pod_key", cmd.PodKey)
	// TODO: Implement prompt sending when handler supports it
}

// handleSubscribeTerminal handles subscribe_terminal command from server.
// This notifies the Runner that a browser wants to observe the terminal via Relay.
// Channel is identified by PodKey (not session ID).
func (c *GRPCConnection) handleSubscribeTerminal(cmd *runnerv1.SubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received subscribe_terminal", "pod_key", cmd.PodKey, "relay_url", cmd.RelayUrl)
	if c.handler == nil {
		log.Warn("No handler set, ignoring subscribe_terminal")
		return
	}

	req := SubscribeTerminalRequest{
		PodKey:          cmd.PodKey,
		RelayURL:        cmd.RelayUrl,
		RunnerToken:     cmd.RunnerToken,
		IncludeSnapshot: cmd.IncludeSnapshot,
		SnapshotHistory: cmd.SnapshotHistory,
	}
	if err := c.handler.OnSubscribeTerminal(req); err != nil {
		log.Error("Failed to subscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleUnsubscribeTerminal handles unsubscribe_terminal command from server.
// This notifies the Runner that all browsers have disconnected.
func (c *GRPCConnection) handleUnsubscribeTerminal(cmd *runnerv1.UnsubscribeTerminalCommand) {
	log := logger.GRPC()
	log.Info("Received unsubscribe_terminal", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring unsubscribe_terminal")
		return
	}

	req := UnsubscribeTerminalRequest{
		PodKey: cmd.PodKey,
	}
	if err := c.handler.OnUnsubscribeTerminal(req); err != nil {
		log.Error("Failed to unsubscribe terminal", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleQuerySandboxes handles query_sandboxes command from server.
// Returns sandbox status for specified pod keys.
func (c *GRPCConnection) handleQuerySandboxes(cmd *runnerv1.QuerySandboxesCommand) {
	log := logger.GRPC()
	log.Info("Received query_sandboxes", "request_id", cmd.RequestId, "queries", len(cmd.Queries))
	if c.handler == nil {
		log.Warn("No handler set, ignoring query_sandboxes")
		return
	}

	req := QuerySandboxesRequest{
		RequestID: cmd.RequestId,
		Queries:   cmd.Queries,
	}
	if err := c.handler.OnQuerySandboxes(req); err != nil {
		log.Error("Failed to query sandboxes", "request_id", cmd.RequestId, "error", err)
	}
}

// handleCreateAutopilot handles create_autopilot command from server.
func (c *GRPCConnection) handleCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) {
	log := logger.GRPC()
	log.Info("Received create_autopilot", "autopilot_key", cmd.AutopilotKey, "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring create_autopilot")
		return
	}

	if err := c.handler.OnCreateAutopilot(cmd); err != nil {
		log.Error("Failed to create Autopilot", "autopilot_key", cmd.AutopilotKey, "error", err)
	}
}

// handleAutopilotControl handles autopilot_control command from server.
func (c *GRPCConnection) handleAutopilotControl(cmd *runnerv1.AutopilotControlCommand) {
	log := logger.GRPC()
	log.Info("Received autopilot_control", "autopilot_key", cmd.AutopilotKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring autopilot_control")
		return
	}

	if err := c.handler.OnAutopilotControl(cmd); err != nil {
		log.Error("Failed to handle Autopilot control", "autopilot_key", cmd.AutopilotKey, "error", err)
	}
}
