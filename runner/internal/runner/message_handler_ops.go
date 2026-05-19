package runner

import (
	"fmt"
	"strings"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ptySubmitGap separates the prompt-body Write from the Enter keystroke so
// the TUI's read(2) loop ticks between them. Without this gap both writes
// land in one read and the TUI treats the whole chunk (incl. trailing \r)
// as a paste — the Enter never fires. MCP's two-RPC path gets this gap
// implicitly via network round-trip; the in-process gRPC path does not.
const ptySubmitGap = 80 * time.Millisecond

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

// OnPodInput handles PTY input from server.
func (h *RunnerMessageHandler) OnPodInput(req client.PodInputRequest) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for PTY input", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}
	if pod.IO == nil {
		log.Warn("PodIO not available for input", "pod_key", req.PodKey)
		return fmt.Errorf("pod IO not available for pod: %s", req.PodKey)
	}
	if err := pod.IO.SendInput(string(req.Data)); err != nil {
		log.Error("Failed to write pod input", "pod_key", req.PodKey, "error", err)
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

// OnObservePod handles observe PTY command from server.
// Reads pod I/O state and sends result back via gRPC.
func (h *RunnerMessageHandler) OnObservePod(req client.ObservePodRequest) error {
	log := logger.Pod()

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		log.Warn("Pod not found for observe PTY", "pod_key", req.PodKey)
		return h.conn.SendObservePodResult(req.RequestID, req.PodKey, "", "", 0, 0, 0, false, "pod not found")
	}

	if pod.IO == nil {
		log.Warn("No PodIO for observe PTY", "pod_key", req.PodKey)
		return h.conn.SendObservePodResult(req.RequestID, req.PodKey, "", "", 0, 0, 0, false, "pod IO not available")
	}

	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}

	output, err := pod.IO.GetSnapshot(lines)
	if err != nil {
		log.Error("Failed to get snapshot for observe PTY", "pod_key", req.PodKey, "error", err)
		return h.conn.SendObservePodResult(req.RequestID, req.PodKey, "", "", 0, 0, 0, false, err.Error())
	}
	var cursorY, cursorX int
	var screen string
	if ta, ok := pod.IO.(TerminalAccess); ok {
		cursorY, cursorX = ta.CursorPosition()
		if req.IncludeScreen {
			screen = ta.GetScreenSnapshot()
		}
	}

	// Count total lines in output to determine hasMore
	totalLines := 0
	if output != "" {
		totalLines = strings.Count(output, "\n") + 1
	}
	hasMore := totalLines >= lines

	if err := h.conn.SendObservePodResult(req.RequestID, req.PodKey, output, screen, cursorX, cursorY, totalLines, hasMore, ""); err != nil {
		log.Error("Failed to send observe PTY result", "request_id", req.RequestID, "error", err)
		return err
	}

	log.Debug("Sent observe PTY result", "request_id", req.RequestID, "pod_key", req.PodKey, "lines", totalLines)
	return nil
}

// OnSendPrompt handles send_prompt command from server (gRPC control plane).
// Mode-transparent submission: ACP submits via its structured SendPrompt RPC;
// PTY writes the body then issues a separate Enter keystroke via SendKeys
// (the "press Enter" semantic — not SendInput which is "raw bytes"). A small
// gap between the two writes is required so the TUI doesn't fold them into
// a single paste.
// For ACP mode, also echoes the user message via Relay so it appears in the
// chat UI (consistent with the Relay command path in handleAcpRelayCommand).
func (h *RunnerMessageHandler) OnSendPrompt(cmd *runnerv1.SendPromptCommand) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(cmd.PodKey)
	if !ok {
		log.Warn("Pod not found for send_prompt", "pod_key", cmd.PodKey)
		return fmt.Errorf("pod not found: %s", cmd.PodKey)
	}
	if pod.IO == nil {
		log.Warn("PodIO not available for send_prompt", "pod_key", cmd.PodKey)
		return fmt.Errorf("pod IO not available: %s", cmd.PodKey)
	}
	// ACP: echo user message to Relay so it appears in the chat panel.
	if pod.IsACPMode() {
		sendAcpViaRelay(pod, "content_chunk", "", map[string]string{
			"text": cmd.Prompt, "role": "user",
		})
	}
	if err := pod.IO.SendInput(cmd.Prompt); err != nil {
		return err
	}
	if ta, ok := pod.IO.(TerminalAccess); ok {
		time.Sleep(ptySubmitGap)
		return ta.SendKeys([]string{"enter"})
	}
	return nil
}
