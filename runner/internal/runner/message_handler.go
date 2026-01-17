package runner

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
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
	log.Printf("[message_handler] Creating pod: pod_key=%s, command=%s, args=%v",
		req.PodKey, req.LaunchCommand, req.LaunchArgs)

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
		WithWorkDirConfig(req.WorkDirConfig).
		WithInitialPrompt(req.InitialPrompt)

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
		log.Printf("[message_handler] Registered pod %s with MCP server (org: %s)", req.PodKey, orgSlug)
	}

	// Send initial prompt if specified
	if req.InitialPrompt != "" {
		time.AfterFunc(1500*time.Millisecond, func() {
			if err := pod.Terminal.Write([]byte(req.InitialPrompt + "\n")); err != nil {
				log.Printf("[message_handler] Failed to send initial prompt: %v", err)
			}
		})
	}

	// Notify server that pod is created
	h.sendPodCreated(req.PodKey, pod.Terminal.PID(), pod.WorktreePath, pod.Branch, 80, 24)

	log.Printf("[message_handler] Pod created: pod_key=%s, pid=%d, worktree=%s, branch=%s",
		req.PodKey, pod.Terminal.PID(), pod.WorktreePath, pod.Branch)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
// Implements client.MessageHandler interface.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log.Printf("[message_handler] Terminating pod: pod_key=%s", req.PodKey)

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

	log.Printf("[message_handler] Pod terminated: pod_key=%s", req.PodKey)
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

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		return fmt.Errorf("failed to decode terminal input: %w", err)
	}

	return pod.Terminal.Write(data)
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
		log.Printf("[message_handler] Pod exited: pod_key=%s, exit_code=%d", podKey, exitCode)

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
		log.Printf("[message_handler] Failed to send pod created event: %v", err)
	}
}

func (h *RunnerMessageHandler) sendPodTerminated(podKey string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with exit code 0 (normal termination)
	if err := h.conn.SendPodTerminated(podKey, 0, ""); err != nil {
		log.Printf("[message_handler] Failed to send pod terminated event: %v", err)
	}
}

func (h *RunnerMessageHandler) sendTerminalOutput(podKey string, data []byte) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method for terminal output
	if err := h.conn.SendTerminalOutput(podKey, data); err != nil {
		log.Printf("[message_handler] Failed to send terminal output: %v", err)
	}
}

func (h *RunnerMessageHandler) sendPtyResized(podKey string, cols, rows uint16) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendPtyResized(podKey, int32(cols), int32(rows)); err != nil {
		log.Printf("[message_handler] Failed to send pty resized event: %v", err)
	}
}

func (h *RunnerMessageHandler) sendPodError(podKey, errorMsg string) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method
	if err := h.conn.SendError(podKey, "error", errorMsg); err != nil {
		log.Printf("[message_handler] Failed to send error event: %v", err)
	}
}

func (h *RunnerMessageHandler) sendPodErrorWithCode(podKey string, podErr *client.PodError) {
	if h.conn == nil {
		return
	}
	// Use gRPC-specific method with error code
	if err := h.conn.SendError(podKey, podErr.Code, podErr.Message); err != nil {
		log.Printf("[message_handler] Failed to send error event: %v", err)
	}
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)
