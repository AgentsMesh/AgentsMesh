package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// sendAcpViaRelay sends an ACP event via the Relay WebSocket.
// If Relay is unavailable, the event is silently dropped (consistent with
// PTY output which also only flows through Relay).
//
// The payload is flat JSON: {"type":"...","session_id":"...","text":"...",...}
// Data struct fields are merged into the top level (not nested under "data").
func sendAcpViaRelay(pod *Pod, eventType, sessionID string, data any) {
	rc := pod.GetRelayClient()
	if rc == nil || !rc.IsConnected() {
		return
	}

	// Marshal data struct to JSON, then merge into top-level map.
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	var flat map[string]any
	if err := json.Unmarshal(dataBytes, &flat); err != nil {
		flat = map[string]any{}
	}
	flat["type"] = eventType
	flat["sessionId"] = sessionID

	payload, err := json.Marshal(flat)
	if err != nil {
		return
	}
	if err := rc.Send(relay.MsgTypeAcpEvent, payload); err != nil {
		// Log critical events that should not be silently lost.
		// Permission requests cause agent to hang if the user never sees them;
		// tool call results leave the UI spinner stuck forever.
		if eventType == "permissionRequest" || eventType == "toolCallResult" || eventType == "log" {
			logger.Pod().Warn("Failed to send critical ACP event via relay",
				"pod_key", pod.PodKey, "event_type", eventType, "error", err)
		}
	}
}

// wireAndStartACPPod creates the ACPClient with Relay-forwarding callbacks,
// wires it into the pod, and starts the subprocess.
func (h *RunnerMessageHandler) wireAndStartACPPod(pod *Pod, cmd *runnerv1.CreatePodCommand, cols, rows int) error {
	log := logger.Pod()
	podKey := cmd.PodKey
	conn := h.conn

	// Pre-declare so callbacks can capture it (NewClient returns the same pointer).
	var acpClient *acp.ACPClient

	// Create ACPClient with event callbacks that forward via Relay.
	acpClient = acp.NewClient(acp.ClientConfig{
		Command:       pod.LaunchCommand,
		Args:          pod.LaunchArgs,
		WorkDir:       pod.WorkDir,
		Env:           pod.LaunchEnv,
		Logger:        log.With("pod_key", podKey),
		TransportType: inferTransportType(pod.LaunchCommand),
		Callbacks: acp.EventCallbacks{
			OnContentChunk: func(sessionID string, chunk acp.ContentChunk) {
				sendAcpViaRelay(pod, "contentChunk", sessionID, chunk)
			},
			OnToolCallUpdate: func(sessionID string, update acp.ToolCallUpdate) {
				sendAcpViaRelay(pod, "toolCallUpdate", sessionID, update)
			},
			OnToolCallResult: func(sessionID string, result acp.ToolCallResult) {
				sendAcpViaRelay(pod, "toolCallResult", sessionID, result)
			},
			OnPlanUpdate: func(sessionID string, update acp.PlanUpdate) {
				sendAcpViaRelay(pod, "planUpdate", sessionID, update)
			},
			OnThinkingUpdate: func(sessionID string, update acp.ThinkingUpdate) {
				sendAcpViaRelay(pod, "thinkingUpdate", sessionID, update)
			},
			OnPermissionRequest: func(req acp.PermissionRequest) {
				// Track pending permission for snapshots.
				acpClient.AddPendingPermission(req)
				sendAcpViaRelay(pod, "permissionRequest", req.SessionID, req)
			},
			OnStateChange: func(newState string) {
				// Lifecycle status update via gRPC (backend updates DB).
				backendStatus := mapACPState(newState)
				_ = conn.SendAgentStatus(podKey, backendStatus)
				// UI state notification via Relay only.
				sendAcpViaRelay(pod, "sessionState", "", map[string]string{"state": newState})
				// Notify PodIO subscribers (e.g. Autopilot StateDetectorCoordinator).
				if sa, ok := pod.IO.(SessionAccess); ok {
					sa.NotifyStateChange(newState)
				}
			},
			OnLog: func(level, message string) {
				sendAcpViaRelay(pod, "log", "", map[string]string{
					"level": level, "message": message,
				})
			},
			OnExit: func(exitCode int) {
				h.handleACPExit(podKey, exitCode)
			},
		},
	})

	// Wire client into pod
	pod.IO = NewACPPodIO(acpClient, podKey)
	pod.Relay = NewACPPodRelay(podKey, acpClient, func(payload []byte) {
		h.handleAcpRelayCommand(pod, payload)
	})

	// Start the ACP client (launches subprocess, performs initialize handshake)
	if err := acpClient.Start(); err != nil {
		h.podStore.Delete(cmd.PodKey)
		if pod.IO != nil {
			pod.IO.Teardown()
		}
		if pod.SandboxPath != "" {
			os.RemoveAll(pod.SandboxPath)
		}
		h.sendPodError(cmd.PodKey, fmt.Sprintf("failed to start ACP agent: %v", err))
		return fmt.Errorf("failed to start ACP agent: %w", err)
	}

	pod.SetStatus(PodStatusRunning)

	// Create a new ACP session with MCP servers config
	mcpPort := h.runner.GetConfig().GetMCPPort()
	mcpServers := acp.BuildMCPServersConfig(mcpPort)
	if err := acpClient.NewSession(mcpServers); err != nil {
		log.Error("Failed to create ACP session", "pod_key", podKey, "error", err)
		// Don't fail pod creation — session can be retried via prompt
	}

	// Send prompt if provided.
	// Claude: sessionID is empty (first message triggers system/init asynchronously).
	// ACP/Codex: sessionID is already set by NewSession().
	// ACPClient.SendPrompt checks State() == Idle (guaranteed after Handshake).
	if cmd.Prompt != "" {
		// Echo user message so it appears in chat on all connected devices.
		sendAcpViaRelay(pod, "contentChunk", "", map[string]string{
			"text": cmd.Prompt, "role": "user",
		})
		if err := acpClient.SendPrompt(cmd.Prompt); err != nil {
			log.Error("Failed to send prompt", "pod_key", podKey, "error", err)
		}
	}

	h.sendPodCreated(cmd.PodKey, 0, pod.SandboxPath, pod.Branch, uint16(cols), uint16(rows))
	log.Info("Pod created (ACP)", "pod_key", cmd.PodKey, "sandbox", pod.SandboxPath)
	return nil
}

// handleACPExit handles ACP subprocess exit.
func (h *RunnerMessageHandler) handleACPExit(podKey string, exitCode int) {
	logger.Pod().Info("ACP process exited", "pod_key", podKey, "exit_code", exitCode)
	h.cleanupPodExit(podKey, exitCode, false)
}

// inferTransportType determines the transport type based on the launch command.
func inferTransportType(command string) string {
	base := strings.TrimSuffix(filepath.Base(command), filepath.Ext(command))
	return acp.TransportTypeForCommand(base)
}
