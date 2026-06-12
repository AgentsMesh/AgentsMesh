package mockagent

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// handleControlRequest implements the server end of the AgentsMesh ACP
// extension defined in //runner/internal/acp/transport_acp.go:SendControlRequest.
//
// Subtype matrix (kept aligned with runner/internal/acp/client_session.go
// callers so any ACPClient.SetXxx / GetXxx / Interrupt round-trip can land
// on the mock without changes):
//
//   set_permission_mode  state.mode ← params.mode
//   set_model            state.model ← params.model
//   set_thinking_level   reserved — accepts + records params.level so
//                        future UI can drive a thinking-level Selector
//   interrupt            best-effort cancel: not enforced on running
//                        scenario goroutines (mock has no thread to
//                        cancel), but acknowledged so the call returns
//                        cleanly
//   get_context_usage    synthetic { input_tokens, output_tokens,
//                        total_tokens } so callers can render usage
//                        without depending on real model output
//
// New subtypes go here. Unknown subtypes return method_not_found so the
// caller's ErrControlNotSupported fall-through still works.
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
	case "set_thinking_level":
		level, _ := req.Params["level"].(string)
		state.setThinkingLevel(level)
		logger.Info("mock set_thinking_level", "level", level)
	case "interrupt":
		logger.Info("mock interrupt acknowledged")
	case "get_context_usage":
		return state.writer.WriteResponse(id, mockContextUsage(), nil)
	default:
		if strings.HasPrefix(req.Subtype, "loopal.") {
			logger.Info("mock loopal control", "subtype", req.Subtype)
			return state.writer.WriteResponse(id, map[string]any{"ok": true}, nil)
		}
		return state.writer.WriteResponse(id, nil, &acp.JSONRPCError{
			Code: acp.ErrCodeMethodNotFound, Message: "unknown subtype: " + req.Subtype,
		})
	}
	return state.writer.WriteResponse(id, map[string]any{"ok": true}, nil)
}

// mockContextUsage returns a stable synthetic usage payload. Tests can
// assert the structure without needing a real LLM to produce numbers.
func mockContextUsage() map[string]any {
	return map[string]any{
		"input_tokens":  1234,
		"output_tokens": 567,
		"total_tokens":  1801,
		"context_window": map[string]any{
			"used":      1801,
			"remaining": 198199,
			"limit":     200000,
		},
	}
}
