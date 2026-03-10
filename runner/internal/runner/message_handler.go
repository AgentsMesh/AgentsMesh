package runner

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// RunnerMessageHandler implements client.MessageHandler interface.
type RunnerMessageHandler struct {
	runner             *Runner
	podStore           PodStore
	conn               client.Connection
	relayClientFactory func(url, podKey, token string, logger *slog.Logger) relay.RelayClient
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner *Runner, store PodStore, conn client.Connection) *RunnerMessageHandler {
	logger.Runner().Debug("Creating message handler")
	return &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
		relayClientFactory: func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
			return relay.NewClient(url, podKey, token, logger)
		},
	}
}

// OnCreatePod handles create pod requests from server.
func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand, "args", cmd.LaunchArgs)

	// Use runner's lifecycle context so long operations (git clone) can be
	// cancelled on shutdown, instead of blocking with context.Background().
	ctx := h.runner.GetRunContext()

	// Register a pending pod placeholder to prevent race conditions:
	// - TerminatePod arriving during Build can find and remove the placeholder
	// - Exit handler after Start can find the pod in store
	h.podStore.Put(cmd.PodKey, &Pod{
		PodKey: cmd.PodKey,
		Status: PodStatusInitializing,
	})

	// Build pod with all components (SRP: PodBuilder handles all component creation)
	cols := int(cmd.Cols)
	rows := int(cmd.Rows)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	builder := NewPodBuilderFromRunner(h.runner).
		WithCommand(cmd).
		WithTerminalSize(cols, rows).
		WithOSCHandler(h.createOSCHandler(cmd.PodKey))

	// Enable PTY logging if configured
	if h.runner.cfg.LogPTY {
		builder.WithPTYLogging(h.runner.cfg.GetLogPTYDir())
	}

	pod, err := builder.Build(ctx)
	if err != nil {
		h.podStore.Delete(cmd.PodKey) // Remove pending placeholder
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Check if pod was terminated during Build (TerminatePod removed the placeholder)
	if _, ok := h.podStore.Get(cmd.PodKey); !ok {
		log.Info("Pod was terminated during build, cleaning up", "pod_key", cmd.PodKey)
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		return fmt.Errorf("pod %s was terminated during build", cmd.PodKey)
	}

	// Set exit handler (callback to MessageHandler for lifecycle events)
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Set PTY error handler to notify frontend when terminal I/O fails.
	// Without this, a PTY read error (e.g., disk full) causes a frozen terminal
	// because the relay stays connected but no data flows through it.
	pod.Terminal.SetPTYErrorHandler(h.createPTYErrorHandler(cmd.PodKey, pod))

	// Replace pending placeholder with fully built pod BEFORE starting terminal.
	// This ensures the exit handler can find the pod if the process exits immediately.
	h.podStore.Put(cmd.PodKey, pod)

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.podStore.Delete(cmd.PodKey) // Remove from store on failure
		// Clean up sandbox that Build() created
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	pod.SetStatus(PodStatusRunning)

	// Register with MCP server and Claude monitor
	if h.runner.mcpServer != nil {
		orgSlug := h.conn.GetOrgSlug()
		h.runner.mcpServer.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
	}
	if h.runner.agentMonitor != nil {
		h.runner.agentMonitor.RegisterPod(cmd.PodKey, pod.Terminal.PID())
	}

	// Subscribe to VT state detection events, bridge to gRPC.
	// Use mutex to protect lastSentStatus since notifySubscribers invokes
	// each callback in a separate goroutine (via safego.Go).
	if pod.VirtualTerminal != nil {
		var statusMu sync.Mutex
		lastSentStatus := ""
		pod.SubscribeStateChange("grpc-agent-status", func(event detector.StateChangeEvent) {
			var backendStatus string
			switch event.NewState {
			case detector.StateExecuting:
				backendStatus = "executing"
			case detector.StateWaiting:
				backendStatus = "waiting"
			case detector.StateNotRunning:
				backendStatus = "idle"
			default:
				return
			}
			statusMu.Lock()
			if backendStatus == lastSentStatus {
				statusMu.Unlock()
				return // Deduplicate
			}
			lastSentStatus = backendStatus
			statusMu.Unlock()
			if err := h.conn.SendAgentStatus(cmd.PodKey, backendStatus); err != nil {
				logger.Pod().Error("Failed to send agent status",
					"pod_key", cmd.PodKey, "status", backendStatus, "error", err)
			}
		})
	}

	h.sendPodCreated(cmd.PodKey, pod.Terminal.PID(), pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))

	log.Info("Pod created", "pod_key", cmd.PodKey, "pid", pod.Terminal.PID(), "sandbox", pod.SandboxPath)
	return nil
}

// OnTerminatePod handles terminate pod requests from server.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	pod := h.podStore.Delete(req.PodKey)
	if pod == nil {
		log.Warn("Pod not found for termination", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	if pod.PTYLogger != nil {
		pod.PTYLogger.Close()
	}
	pod.StopStateDetector()
	// Stop aggregator BEFORE disconnecting relay, so the final flush
	// can still be sent through the relay (matches createExitHandler order).
	if pod.Aggregator != nil {
		pod.Aggregator.Stop()
	}
	pod.DisconnectRelay()
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}

	if h.runner.mcpServer != nil {
		h.runner.mcpServer.UnregisterPod(req.PodKey)
	}
	if h.runner.agentMonitor != nil {
		h.runner.agentMonitor.UnregisterPod(req.PodKey)
	}

	h.sendPodTerminated(req.PodKey)
	log.Info("Pod terminated", "pod_key", req.PodKey)
	return nil
}

// OnListPods returns current pods.
func (h *RunnerMessageHandler) OnListPods() []client.PodInfo {
	pods := h.podStore.All()
	result := make([]client.PodInfo, 0, len(pods))

	for _, s := range pods {
		info := client.PodInfo{
			PodKey:      s.PodKey,
			Status:      s.GetStatus(),
			AgentStatus: h.getAgentStatusFromDetector(s),
		}
		if s.Terminal != nil {
			info.Pid = s.Terminal.PID()
		}
		result = append(result, info)
	}

	return result
}

// getAgentStatusFromDetector maps the detector's AgentState to backend status string.
func (h *RunnerMessageHandler) getAgentStatusFromDetector(pod *Pod) string {
	if pod.VirtualTerminal == nil {
		return "idle"
	}
	d := pod.GetOrCreateStateDetector()
	if d == nil {
		return "idle"
	}
	switch d.GetState() {
	case detector.StateExecuting:
		return "executing"
	case detector.StateWaiting:
		return "waiting"
	case detector.StateNotRunning:
		return "idle"
	default:
		return "idle"
	}
}

// OnListRelayConnections returns current relay connections.
func (h *RunnerMessageHandler) OnListRelayConnections() []client.RelayConnectionInfo {
	pods := h.podStore.All()
	result := make([]client.RelayConnectionInfo, 0)

	for _, pod := range pods {
		relayClient := pod.GetRelayClient()
		if relayClient != nil {
			result = append(result, client.RelayConnectionInfo{
				PodKey:      pod.PodKey,
				RelayURL:    relayClient.GetRelayURL(),
				Connected:   relayClient.IsConnected(),
				ConnectedAt: relayClient.GetConnectedAt(),
			})
		}
	}

	return result
}

// OnTerminalInput handles terminal input from server.
func (h *RunnerMessageHandler) OnTerminalInput(req client.TerminalInputRequest) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for terminal input", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}
	if pod.Terminal == nil {
		log.Warn("Terminal not initialized for input", "pod_key", req.PodKey)
		return fmt.Errorf("terminal not initialized for pod: %s", req.PodKey)
	}
	if err := pod.Terminal.Write(req.Data); err != nil {
		log.Error("Failed to write terminal input", "pod_key", req.PodKey, "error", err)
		return err
	}
	return nil
}

// OnTerminalResize handles terminal resize requests from server.
func (h *RunnerMessageHandler) OnTerminalResize(req client.TerminalResizeRequest) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for terminal resize", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	if err := pod.Terminal.Resize(int(req.Cols), int(req.Rows)); err != nil {
		log.Error("Failed to resize terminal", "pod_key", req.PodKey, "cols", req.Cols, "rows", req.Rows, "error", err)
		return err
	}
	if pod.VirtualTerminal != nil {
		pod.VirtualTerminal.Resize(int(req.Cols), int(req.Rows))
	}

	log.Debug("Terminal resized", "pod_key", req.PodKey, "cols", req.Cols, "rows", req.Rows)
	h.sendPtyResized(req.PodKey, req.Cols, req.Rows)
	return nil
}

// OnTerminalRedraw handles terminal redraw requests from server.
func (h *RunnerMessageHandler) OnTerminalRedraw(req client.TerminalRedrawRequest) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for terminal redraw", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}
	log.Info("Triggering terminal redraw", "pod_key", req.PodKey)
	if err := pod.Terminal.Redraw(); err != nil {
		log.Error("Failed to redraw terminal", "pod_key", req.PodKey, "error", err)
		return err
	}
	return nil
}

// Note: OnSubscribeTerminal, setupRelayClientHandlers, OnUnsubscribeTerminal are in message_handler_relay.go

// OnQuerySandboxes handles sandbox status query from server.
func (h *RunnerMessageHandler) OnQuerySandboxes(req client.QuerySandboxesRequest) error {
	log := logger.Pod()
	log.Info("Querying sandbox status", "request_id", req.RequestID, "queries", len(req.Queries))

	results := make([]*client.SandboxStatusInfo, 0, len(req.Queries))
	for _, query := range req.Queries {
		status := h.runner.GetSandboxStatus(query.PodKey)
		results = append(results, status)
	}

	if err := h.conn.SendSandboxesStatus(req.RequestID, results); err != nil {
		log.Error("Failed to send sandbox status response", "request_id", req.RequestID, "error", err)
		return err
	}

	log.Info("Sent sandbox status response", "request_id", req.RequestID, "results", len(results))
	return nil
}

// Ensure RunnerMessageHandler implements client.MessageHandler
var _ client.MessageHandler = (*RunnerMessageHandler)(nil)
