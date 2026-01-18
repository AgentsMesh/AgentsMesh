package runner

import (
	"context"
	"fmt"

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
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnCreatePod(req client.CreatePodRequest) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", req.PodKey, "command", req.LaunchCommand, "args", req.LaunchArgs)

	ctx := context.Background()

	// Check capacity
	if h.runner.cfg.MaxConcurrentPods > 0 && h.podStore.Count() >= h.runner.cfg.MaxConcurrentPods {
		h.sendPodError(req.PodKey, "max concurrent pods reached")
		return fmt.Errorf("max concurrent pods reached")
	}

	// Build pod using new protocol
	builder := NewPodBuilder(h.runner).
		WithPodKey(req.PodKey).
		WithNewProtocol(true).
		WithLaunchCommand(req.LaunchCommand, req.LaunchArgs).
		WithEnvVars(req.EnvVars).
		WithFilesToCreate(req.FilesToCreate).
		WithWorkDirConfig(req.WorkDirConfig)

	// Build pod
	pod, err := builder.Build(ctx)
	if err != nil {
		// Check if it's a PodError with error code
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(req.PodKey, podErr)
		} else {
			h.sendPodError(req.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Set output/exit handlers
	pod.Terminal.SetOutputHandler(h.createOutputHandler(req.PodKey))
	pod.Terminal.SetExitHandler(h.createExitHandler(req.PodKey))

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.sendPodError(req.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	// Store pod
	h.podStore.Put(req.PodKey, pod)
	pod.SetStatus(PodStatusRunning)

	// Register pod with MCP HTTP Server for tool access
	if h.runner.mcpServer != nil {
		orgSlug := h.conn.GetOrgSlug()
		h.runner.mcpServer.RegisterPod(req.PodKey, orgSlug, nil, nil, req.LaunchCommand)
		log.Debug("Registered pod with MCP server", "pod_key", req.PodKey, "org", orgSlug)
	}

	// Notify server that pod is created
	h.sendPodCreated(req.PodKey, pod.Terminal.PID(), pod.WorktreePath, pod.Branch, 80, 24)

	log.Info("Pod created", "pod_key", req.PodKey, "pid", pod.Terminal.PID(), "worktree", pod.WorktreePath, "branch", pod.Branch)
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
