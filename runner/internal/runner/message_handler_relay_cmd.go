package runner

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// handleAcpRelayCommand parses and routes an ACP command received via Relay.
// Payload format from frontend: {"type":"prompt","prompt":"..."} (flat JSON).
func (h *RunnerMessageHandler) handleAcpRelayCommand(pod *Pod, payload []byte) {
	log := logger.Pod()
	var cmd struct {
		Type   string `json:"type"`
		Prompt string `json:"prompt"`    // prompt command
		ReqID  string `json:"requestId"` // permission_response
		Approv bool   `json:"approved"`  // permission_response
	}
	if err := json.Unmarshal(payload, &cmd); err != nil {
		log.Warn("Failed to parse ACP relay command", "pod_key", pod.PodKey, "error", err)
		return
	}

	if pod.IO == nil {
		log.Warn("Pod IO not available for ACP command", "pod_key", pod.PodKey)
		return
	}

	sa, ok := pod.IO.(SessionAccess)

	switch cmd.Type {
	case "prompt":
		// Echo user message back to all relay subscribers so it appears in chat.
		sendAcpViaRelay(pod, "contentChunk", "", map[string]string{
			"text": cmd.Prompt, "role": "user",
		})
		if err := pod.IO.SendInput(cmd.Prompt); err != nil {
			log.Error("Failed to send ACP prompt via relay", "pod_key", pod.PodKey, "error", err)
		}

	case "permission_response":
		if !ok {
			log.Warn("SessionAccess not available for permission_response", "pod_key", pod.PodKey)
			return
		}
		if err := sa.RespondToPermission(cmd.ReqID, cmd.Approv); err != nil {
			log.Error("Failed to respond to ACP permission via relay", "pod_key", pod.PodKey, "error", err)
		}

	case "cancel":
		if !ok {
			log.Warn("SessionAccess not available for cancel", "pod_key", pod.PodKey)
			return
		}
		if err := sa.CancelSession(); err != nil {
			log.Error("Failed to cancel ACP session via relay", "pod_key", pod.PodKey, "error", err)
		}

	default:
		log.Debug("Unknown ACP relay command type", "pod_key", pod.PodKey, "type", cmd.Type)
	}
}
