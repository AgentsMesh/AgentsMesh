package mockagent

import (
	"encoding/json"
	"log/slog"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// handleControlRequest implements the server end of the AgentsMesh ACP
// extension defined in //runner/internal/acp/transport_acp.go:SendControlRequest.
// Each subtype mutates runtime state (so subsequent get_* queries reflect it)
// and answers with `{ok: true}` so the runner's OnConfigChange callback fires.
//
// New subtypes go here. The runner won't ship a new control method until at
// least one agent (mock or otherwise) implements it; this central switch is
// the contract.
func handleControlRequest(state *runtimeState, id int64, raw json.RawMessage, logger *slog.Logger) error {
	var req struct {
		SessionID string         `json:"sessionId"`
		Subtype   string         `json:"subtype"`
		Params    map[string]any `json:"params"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return state.writer.WriteResponse(id, nil, &acp.JSONRPCError{
			Code: acp.ErrCodeInvalidParams, Message: err.Error(),
		})
	}
	switch req.Subtype {
	case "set_permission_mode":
		mode, _ := req.Params["mode"].(string)
		state.setPermissionMode(mode)
		logger.Info("mock set_permission_mode", "mode", mode)
	case "set_model":
		model, _ := req.Params["model"].(string)
		state.setModel(model)
		logger.Info("mock set_model", "model", model)
	default:
		return state.writer.WriteResponse(id, nil, &acp.JSONRPCError{
			Code: acp.ErrCodeMethodNotFound, Message: "unknown subtype: " + req.Subtype,
		})
	}
	return state.writer.WriteResponse(id, map[string]any{"ok": true}, nil)
}
