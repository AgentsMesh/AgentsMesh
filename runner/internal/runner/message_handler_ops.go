package runner

import (
	"fmt"

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
