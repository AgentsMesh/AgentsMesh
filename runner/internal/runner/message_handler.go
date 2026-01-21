package runner

import (
	"context"
	"fmt"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// RunnerMessageHandler implements client.MessageHandler interface.
// It bridges the client protocol layer with the runner pod management.
type RunnerMessageHandler struct {
	runner   *Runner
	podStore PodStore
	conn     client.Connection
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner *Runner, store PodStore, conn client.Connection) *RunnerMessageHandler {
	return &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
	}
}

// OnCreatePod handles create pod requests from server.
// Uses Proto type directly for zero-copy message passing.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand, "args", cmd.LaunchArgs)

	ctx := context.Background()

	// Check capacity
	if h.runner.cfg.MaxConcurrentPods > 0 && h.podStore.Count() >= h.runner.cfg.MaxConcurrentPods {
		h.sendPodError(cmd.PodKey, "max concurrent pods reached")
		return fmt.Errorf("max concurrent pods reached")
	}

	// Build pod using Proto command directly
	builder := NewPodBuilder(h.runner).
		WithCommand(cmd)

	// Build pod
	pod, err := builder.Build(ctx)
	if err != nil {
		// Check if it's a PodError with error code
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Set output/exit handlers
	pod.Terminal.SetOutputHandler(h.createOutputHandler(cmd.PodKey))
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	// Store pod
	h.podStore.Put(cmd.PodKey, pod)
	pod.SetStatus(PodStatusRunning)

	// Register pod with MCP HTTP Server for tool access
	if h.runner.mcpServer != nil {
		orgSlug := h.conn.GetOrgSlug()
		h.runner.mcpServer.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
		log.Debug("Registered pod with MCP server", "pod_key", cmd.PodKey, "org", orgSlug)
	}

	// Notify server that pod is created
	h.sendPodCreated(cmd.PodKey, pod.Terminal.PID(), pod.WorktreePath, pod.Branch, 80, 24)

	log.Info("Pod created", "pod_key", cmd.PodKey, "pid", pod.Terminal.PID(), "worktree", pod.WorktreePath, "branch", pod.Branch)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	pod := h.podStore.Delete(req.PodKey)
	if pod == nil {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	// Stop terminal
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}

	// Unregister pod from MCP HTTP Server
	if h.runner.mcpServer != nil {
		h.runner.mcpServer.UnregisterPod(req.PodKey)
	}

	// Notify server
	h.sendPodTerminated(req.PodKey)

	log.Info("Pod terminated", "pod_key", req.PodKey)
	return nil
}

// OnListPods returns current pods.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnListPods() []client.PodInfo {
	pods := h.podStore.All()
	result := make([]client.PodInfo, 0, len(pods))

	for _, s := range pods {
		info := client.PodInfo{
			PodKey:       s.PodKey,
			Status:       s.GetStatus(),
			ClaudeStatus: "",
		}
		if s.Terminal != nil {
			info.Pid = s.Terminal.PID()
		}
		result = append(result, info)
	}

	return result
}

// OnTerminalInput handles terminal input from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalInput(req client.TerminalInputRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	if pod.Terminal == nil {
		return fmt.Errorf("terminal not initialized for pod: %s", req.PodKey)
	}

	// gRPC uses native bytes, no decoding needed
	return pod.Terminal.Write(req.Data)
}

// OnTerminalResize handles terminal resize requests from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalResize(req client.TerminalResizeRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	if err := pod.Terminal.Resize(int(req.Rows), int(req.Cols)); err != nil {
		return err
	}

	// Notify server of resize
	h.sendPtyResized(req.PodKey, req.Cols, req.Rows)
	return nil
}

// OnTerminalRedraw handles terminal redraw requests from server.
// Uses resize +1/-1 trick to trigger terminal redraw (SIGWINCH alone doesn't work
// for programs in idle state like Claude Code waiting for input).
// Used for restoring terminal state after server restart.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminalRedraw(req client.TerminalRedrawRequest) error {
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	logger.Pod().Info("Triggering terminal redraw", "pod_key", req.PodKey)

	// Use resize trick to trigger redraw - no pty_resized event sent back
	return pod.Terminal.Redraw()
}

// Helper methods

func (h *RunnerMessageHandler) createOutputHandler(podKey string) func([]byte) {
	return func(data []byte) {
		h.sendTerminalOutput(podKey, data)
	}
}

func (h *RunnerMessageHandler) createExitHandler(podKey string) func(int) {
	return func(exitCode int) {
		logger.Pod().Info("Pod exited", "pod_key", podKey, "exit_code", exitCode)

		pod := h.podStore.Delete(podKey)
		if pod != nil {
			pod.SetStatus(PodStatusStopped)
		}

		h.sendPodTerminated(podKey)
	}
}

// Event sending methods - using gRPC-specific methods

func (h *RunnerMessageHandler) sendPodCreated(podKey string, pid int, worktreePath, branchName string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendPodCreated(podKey, int32(pid)); err != nil {
		logger.Pod().Error("Failed to send pod created event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodTerminated(podKey string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with exit code 0 (normal termination)
	if err := h.conn.SendPodTerminated(podKey, 0, ""); err != nil {
		logger.Pod().Error("Failed to send pod terminated event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendTerminalOutput(podKey string, data []byte) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method for terminal output
	if err := h.conn.SendTerminalOutput(podKey, data); err != nil {
		logger.Terminal().Error("Failed to send terminal output", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPtyResized(podKey string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendPtyResized(podKey, int32(cols), int32(rows)); err != nil {
		logger.Terminal().Error("Failed to send pty resized event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodError(podKey, errorMsg string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendError(podKey, "error", errorMsg); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}

func (h *RunnerMessageHandler) sendPodErrorWithCode(podKey string, podErr *client.PodError) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with error code
	if err := h.conn.SendError(podKey, podErr.Code, podErr.Message); err != nil {
		logger.Pod().Error("Failed to send error event", "error", err)
	}
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)
