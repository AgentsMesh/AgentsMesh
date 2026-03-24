package runner

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

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

	// Adapt input for agent-specific TUI requirements
	data := adaptTerminalInput(req.Data, pod.AgentType)

	if err := pod.Terminal.Write(data); err != nil {
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

	if pod.Terminal == nil {
		log.Warn("Terminal not initialized for resize", "pod_key", req.PodKey)
		return fmt.Errorf("terminal not initialized for pod: %s", req.PodKey)
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
	if pod.Terminal == nil {
		log.Warn("Terminal not initialized for redraw", "pod_key", req.PodKey)
		return fmt.Errorf("terminal not initialized for pod: %s", req.PodKey)
	}
	log.Info("Triggering terminal redraw", "pod_key", req.PodKey)
	if err := pod.Terminal.Redraw(); err != nil {
		log.Error("Failed to redraw terminal", "pod_key", req.PodKey, "error", err)
		return err
	}
	return nil
}

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

// OnObserveTerminal handles observe terminal command from server.
// Reads VirtualTerminal state and sends result back via gRPC.
func (h *RunnerMessageHandler) OnObserveTerminal(req client.ObserveTerminalRequest) error {
	log := logger.Pod()

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for observe terminal", "pod_key", req.PodKey)
		return h.conn.SendObserveTerminalResult(req.RequestID, req.PodKey, "", "", 0, 0, 0, false, "pod not found")
	}

	if pod.VirtualTerminal == nil {
		log.Warn("No virtual terminal for observe", "pod_key", req.PodKey)
		return h.conn.SendObserveTerminalResult(req.RequestID, req.PodKey, "", "", 0, 0, 0, false, "no virtual terminal")
	}

	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}

	output := pod.VirtualTerminal.GetOutput(lines)
	cursorY, cursorX := pod.VirtualTerminal.CursorPosition()

	var screen string
	if req.IncludeScreen {
		screen = pod.VirtualTerminal.GetScreenSnapshot()
	}

	// Count total lines in output to determine hasMore
	totalLines := 0
	if output != "" {
		totalLines = strings.Count(output, "\n") + 1
	}
	hasMore := totalLines >= lines

	if err := h.conn.SendObserveTerminalResult(req.RequestID, req.PodKey, output, screen, cursorX, cursorY, totalLines, hasMore, ""); err != nil {
		log.Error("Failed to send observe terminal result", "request_id", req.RequestID, "error", err)
		return err
	}

	log.Debug("Sent observe terminal result", "request_id", req.RequestID, "pod_key", req.PodKey, "lines", totalLines)
	return nil
}
