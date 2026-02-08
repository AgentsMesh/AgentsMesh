package runner

import (
	"context"
	"fmt"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// RunnerMessageHandler implements client.MessageHandler interface.
type RunnerMessageHandler struct {
	runner   *Runner
	podStore PodStore
	conn     client.Connection
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner *Runner, store PodStore, conn client.Connection) *RunnerMessageHandler {
	logger.Runner().Debug("Creating message handler")
	return &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
	}
}

// OnCreatePod handles create pod requests from server.
func (h *RunnerMessageHandler) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	log := logger.Pod()
	log.Info("Creating pod", "pod_key", cmd.PodKey, "command", cmd.LaunchCommand, "args", cmd.LaunchArgs)

	ctx := context.Background()

	// Check capacity
	if h.runner.cfg.MaxConcurrentPods > 0 && h.podStore.Count() >= h.runner.cfg.MaxConcurrentPods {
		h.sendPodError(cmd.PodKey, "max concurrent pods reached")
		return fmt.Errorf("max concurrent pods reached")
	}

	// Build pod
	cols := int(cmd.Cols)
	rows := int(cmd.Rows)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	builder := NewPodBuilder(h.runner).
		WithCommand(cmd).
		WithTerminalSize(cols, rows)

	pod, err := builder.Build(ctx)
	if err != nil {
		if podErr, ok := err.(*client.PodError); ok {
			h.sendPodErrorWithCode(cmd.PodKey, podErr)
		} else {
			h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to build pod: %v", err))
		}
		return fmt.Errorf("failed to build pod: %w", err)
	}

	// Create VirtualTerminal
	pod.VirtualTerminal = NewVirtualTerminal(cols, rows, 100)

	// Create SmartAggregator
	podKey := cmd.PodKey
	vt := pod.VirtualTerminal
	vt.SetOSCHandler(h.createOSCHandler(podKey))

	aggregator := terminal.NewSmartAggregator(nil, nil,
		terminal.WithFullRedrawThrottling(),
	)
	pod.Aggregator = aggregator

	// Set up PTY logging if enabled
	if h.runner.cfg.LogPTY {
		ptyLogger, err := terminal.NewPTYLogger(h.runner.cfg.GetLogPTYDir(), cmd.PodKey)
		if err != nil {
			log.Warn("Failed to create PTY logger", "pod_key", cmd.PodKey, "error", err)
		} else {
			aggregator.SetPTYLogger(ptyLogger)
			pod.PTYLogger = ptyLogger
			log.Info("PTY logging enabled for pod", "pod_key", cmd.PodKey, "log_dir", ptyLogger.LogDir())
		}
	}

	// Set output handler
	pod.Terminal.SetOutputHandler(func(data []byte) {
		defer func() {
			if r := recover(); r != nil {
				logger.Terminal().Error("PANIC in OutputHandler recovered",
					"pod_key", podKey,
					"panic", fmt.Sprintf("%v", r),
					"data_len", len(data))
			}
		}()

		var screenLines []string
		if vt != nil {
			screenLines = vt.Feed(data)
		}
		go pod.NotifyStateDetectorWithScreen(len(data), screenLines)
		aggregator.Write(data)
	})
	pod.Terminal.SetExitHandler(h.createExitHandler(cmd.PodKey))

	// Start terminal
	if err := pod.Terminal.Start(); err != nil {
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start terminal: %v", err))
		return fmt.Errorf("failed to start terminal: %w", err)
	}

	// Store pod
	h.podStore.Put(cmd.PodKey, pod)
	pod.SetStatus(PodStatusRunning)

	// Register with MCP server and Claude monitor
	if h.runner.mcpServer != nil {
		orgSlug := h.conn.GetOrgSlug()
		h.runner.mcpServer.RegisterPod(cmd.PodKey, orgSlug, nil, nil, cmd.LaunchCommand)
	}
	if h.runner.claudeMonitor != nil {
		h.runner.claudeMonitor.RegisterPod(cmd.PodKey, pod.Terminal.PID())
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
	if pod.Aggregator != nil {
		pod.Aggregator.Stop()
	}
	if pod.Terminal != nil {
		pod.Terminal.Stop()
	}

	if h.runner.mcpServer != nil {
		h.runner.mcpServer.UnregisterPod(req.PodKey)
	}
	if h.runner.claudeMonitor != nil {
		h.runner.claudeMonitor.UnregisterPod(req.PodKey)
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
