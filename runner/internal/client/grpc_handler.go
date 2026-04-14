// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
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
			// Don't update lastRecvTime on error — only track successful receives
			if err == io.EOF {
				log.Info("Stream ended (EOF)")
				return
			}
			if status.Code(err) == codes.Canceled {
				logger.GRPCTrace().Trace("Stream cancelled")
			} else if fatal, hint := isFatalStreamError(err); fatal {
				log.Error("Fatal stream error (will not retry)", "error", err)
				log.Error(hint)
				c.setFatalError(fmt.Errorf("%s", hint))
			} else {
				log.Error("Stream error", "error", err)
			}
			return
		}
		// Record successful recv for liveness tracking and diagnostics
		c.lastRecvTime.Store(time.Now().UnixNano())
		c.handleServerMessage(msg)
	}
}

// handleServerMessage dispatches received server messages to appropriate handlers.
// Heavy operations (CreatePod, SubscribePod, CreateAutopilot) are dispatched
// asynchronously via goroutines to avoid blocking the readLoop.
// Lightweight operations remain synchronous to preserve message ordering.
func (c *GRPCConnection) handleServerMessage(msg *runnerv1.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.ServerMessage_InitializeResult:
		c.handleInitializeResult(payload.InitializeResult)

	// Heavy operations - dispatched via per-pod command queue.
	// Same pod's commands execute sequentially (create_pod before create_autopilot).
	// Different pods execute concurrently. Tracked by handlerWg for clean shutdown.
	case *runnerv1.ServerMessage_CreatePod:
		c.handlerWg.Add(1)
		c.podQueue.Enqueue(payload.CreatePod.PodKey, func() {
			defer c.handlerWg.Done()
			c.handleCreatePod(payload.CreatePod)
		})

	case *runnerv1.ServerMessage_TerminatePod:
		c.handlerWg.Add(1)
		c.podQueue.Enqueue(payload.TerminatePod.PodKey, func() {
			defer c.handlerWg.Done()
			c.handleTerminatePod(payload.TerminatePod)
			c.podQueue.Remove(payload.TerminatePod.PodKey)
		})

	case *runnerv1.ServerMessage_SubscribePod:
		c.handlerWg.Add(1)
		go func() {
			defer c.handlerWg.Done()
			c.handleSubscribePod(payload.SubscribePod)
		}()

	case *runnerv1.ServerMessage_CreateAutopilot:
		c.handlerWg.Add(1)
		c.podQueue.Enqueue(payload.CreateAutopilot.PodKey, func() {
			defer c.handlerWg.Done()
			c.handleCreateAutopilot(payload.CreateAutopilot)
		})

	// Lightweight operations - synchronous to preserve ordering
	case *runnerv1.ServerMessage_PodInput:
		c.handlePodInput(payload.PodInput)

	case *runnerv1.ServerMessage_SendPrompt:
		c.handleSendPrompt(payload.SendPrompt)

	case *runnerv1.ServerMessage_UnsubscribePod:
		c.handleUnsubscribePod(payload.UnsubscribePod)

	case *runnerv1.ServerMessage_QuerySandboxes:
		c.handleQuerySandboxes(payload.QuerySandboxes)

	case *runnerv1.ServerMessage_ObservePod:
		c.handleObservePod(payload.ObservePod)

	case *runnerv1.ServerMessage_AutopilotControl:
		c.handleAutopilotControl(payload.AutopilotControl)

	case *runnerv1.ServerMessage_McpResponse:
		c.handleMcpResponse(payload.McpResponse)

	case *runnerv1.ServerMessage_Ping:
		c.handlePing(payload.Ping)

	case *runnerv1.ServerMessage_HeartbeatAck:
		c.handleHeartbeatAck(payload.HeartbeatAck)

	case *runnerv1.ServerMessage_UpgradeRunner:
		c.handlerWg.Add(1)
		safego.Go("handle-upgrade-runner", func() {
			defer c.handlerWg.Done()
			c.handleUpgradeRunner(payload.UpgradeRunner)
		})

	case *runnerv1.ServerMessage_UploadLogs:
		c.handlerWg.Add(1)
		safego.Go("handle-upload-logs", func() {
			defer c.handlerWg.Done()
			c.handleUploadLogs(payload.UploadLogs)
		})

	case *runnerv1.ServerMessage_UpdatePodPerpetual:
		c.handleUpdatePodPerpetual(payload.UpdatePodPerpetual)

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

// Note: Terminal, subscription, autopilot, MCP, and heartbeat handlers
// are in grpc_handler_dispatch.go

// handleUpgradeRunner handles upgrade_runner command from server.
func (c *GRPCConnection) handleUpgradeRunner(cmd *runnerv1.UpgradeRunnerCommand) {
	log := logger.GRPC()
	log.Info("Received upgrade_runner", "request_id", cmd.RequestId, "target_version", cmd.TargetVersion)
	if c.handler == nil {
		log.Warn("No handler set, ignoring upgrade_runner")
		return
	}

	if err := c.handler.OnUpgradeRunner(cmd); err != nil {
		log.Error("Failed to handle upgrade runner", "request_id", cmd.RequestId, "error", err)
	}
}

// handleUpdatePodPerpetual handles update_pod_perpetual command from server.
func (c *GRPCConnection) handleUpdatePodPerpetual(cmd *runnerv1.UpdatePodPerpetualCommand) {
	log := logger.GRPC()
	log.Info("Received update_pod_perpetual", "pod_key", cmd.PodKey, "perpetual", cmd.Perpetual)
	if c.handler == nil {
		log.Warn("No handler set, ignoring update_pod_perpetual")
		return
	}
	if err := c.handler.OnUpdatePodPerpetual(cmd); err != nil {
		log.Error("Failed to update pod perpetual", "pod_key", cmd.PodKey, "error", err)
	}
}
