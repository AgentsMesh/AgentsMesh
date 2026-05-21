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
		Type         string         `json:"type"`
		Prompt       string         `json:"prompt"`       // prompt command
		ReqID        string         `json:"requestId"`    // permission_response
		Approved     bool           `json:"approved"`     // permission_response
		UpdatedInput map[string]any `json:"updatedInput"` // permission_response (AskUserQuestion answers)
		Mode         string         `json:"mode"`          // set_permission_mode
		Model        string         `json:"model"`         // set_model
		Subtype      string         `json:"subtype"`       // control_request (generic)
		Payload      map[string]any `json:"payload"`       // control_request (generic)
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
		if err := sa.RespondToPermission(cmd.ReqID, cmd.Approved, cmd.UpdatedInput); err != nil {
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

	case "interrupt":
		if !ok {
			log.Warn("SessionAccess not available for interrupt", "pod_key", pod.PodKey)
			return
		}
		if err := sa.Interrupt(); err != nil {
			log.Error("Failed to interrupt ACP session", "pod_key", pod.PodKey, "error", err)
		}

	case "set_permission_mode":
		if !ok {
			log.Warn("SessionAccess not available for set_permission_mode", "pod_key", pod.PodKey)
			return
		}
		if err := sa.SetPermissionMode(cmd.Mode); err != nil {
			log.Error("Failed to set permission mode", "pod_key", pod.PodKey, "mode", cmd.Mode, "error", err)
			// Tell the UI the change did NOT take effect. Without this, the
			// post-Phase-D Selector waits forever for a configChanged
			// broadcast that will never arrive and looks frozen.
			sendAcpViaRelay(pod, "configChangeFailed", "", map[string]string{
				"field":   "permissionMode",
				"value":   cmd.Mode,
				"message": err.Error(),
			})
		}

	case "set_model":
		if !ok {
			log.Warn("SessionAccess not available for set_model", "pod_key", pod.PodKey)
			return
		}
		if err := sa.SetModel(cmd.Model); err != nil {
			log.Error("Failed to set model", "pod_key", pod.PodKey, "model", cmd.Model, "error", err)
			sendAcpViaRelay(pod, "configChangeFailed", "", map[string]string{
				"field":   "model",
				"value":   cmd.Model,
				"message": err.Error(),
			})
		}

	case "control_request":
		if !ok {
			log.Warn("SessionAccess not available for control_request", "pod_key", pod.PodKey)
			return
		}
		resp, err := sa.SendControlRequest(cmd.Subtype, cmd.Payload)
		if err != nil {
			log.Error("Failed to send control_request", "pod_key", pod.PodKey, "subtype", cmd.Subtype, "error", err)
		}
		// Forward response back via relay if needed.
		if resp != nil {
			sendAcpViaRelay(pod, "controlResponse", "", resp)
		}

	default:
		log.Debug("Unknown ACP relay command type", "pod_key", pod.PodKey, "type", cmd.Type)
	}
}
