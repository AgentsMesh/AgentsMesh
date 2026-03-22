package acp

import (
	"encoding/json"
	"log/slog"
)

// Handler dispatches inbound JSON-RPC notifications from the agent.
type Handler struct {
	callbacks EventCallbacks
	logger    *slog.Logger
}

// NewHandler creates a Handler that routes notifications to the
// provided callbacks.
func NewHandler(callbacks EventCallbacks, logger *slog.Logger) *Handler {
	return &Handler{callbacks: callbacks, logger: logger}
}

// HandleNotification processes an inbound notification from the agent.
func (h *Handler) HandleNotification(method string, params json.RawMessage) {
	switch method {
	case "session/update":
		h.handleSessionUpdate(params)
	case "session/complete":
		h.handleSessionComplete(params)
	case "permission/request":
		h.handlePermissionRequest(params)
	default:
		h.logger.Debug("unhandled notification", "method", method)
	}
}

func (h *Handler) handleSessionUpdate(params json.RawMessage) {
	var raw struct {
		SessionID string          `json:"session_id"`
		Type      string          `json:"type"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(params, &raw); err != nil {
		h.logger.Warn("failed to parse session/update", "error", err)
		return
	}

	switch raw.Type {
	case "content":
		h.handleContentUpdate(raw.SessionID, raw.Data)
	case "tool_call":
		h.handleToolCallUpdate(raw.SessionID, raw.Data)
	case "tool_result":
		h.handleToolResultUpdate(raw.SessionID, raw.Data)
	case "plan":
		h.handlePlanUpdate(raw.SessionID, raw.Data)
	case "thinking":
		h.handleThinkingUpdate(raw.SessionID, raw.Data)
	default:
		h.logger.Debug("unhandled session/update type", "type", raw.Type)
	}
}

func (h *Handler) handleContentUpdate(sessionID string, data json.RawMessage) {
	var chunk struct {
		Text string `json:"text"`
		Role string `json:"role"`
	}
	if err := json.Unmarshal(data, &chunk); err != nil {
		h.logger.Warn("failed to parse content update", "error", err)
		return
	}
	if h.callbacks.OnContentChunk != nil {
		h.callbacks.OnContentChunk(sessionID, ContentChunk{
			Text: chunk.Text, Role: chunk.Role,
		})
	}
}

func (h *Handler) handleToolCallUpdate(sessionID string, data json.RawMessage) {
	var tc struct {
		ToolCallID    string `json:"tool_call_id"`
		ToolName      string `json:"tool_name"`
		Status        string `json:"status"`
		ArgumentsJSON string `json:"arguments_json"`
	}
	if err := json.Unmarshal(data, &tc); err != nil {
		h.logger.Warn("failed to parse tool_call update", "error", err)
		return
	}
	if h.callbacks.OnToolCallUpdate != nil {
		h.callbacks.OnToolCallUpdate(sessionID, ToolCallUpdate{
			ToolCallID:    tc.ToolCallID,
			ToolName:      tc.ToolName,
			Status:        tc.Status,
			ArgumentsJSON: tc.ArgumentsJSON,
		})
	}
}

func (h *Handler) handleToolResultUpdate(sessionID string, data json.RawMessage) {
	var tr struct {
		ToolCallID   string `json:"tool_call_id"`
		ToolName     string `json:"tool_name"`
		Success      bool   `json:"success"`
		ResultText   string `json:"result_text"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal(data, &tr); err != nil {
		h.logger.Warn("failed to parse tool_result update", "error", err)
		return
	}
	if h.callbacks.OnToolCallResult != nil {
		h.callbacks.OnToolCallResult(sessionID, ToolCallResult{
			ToolCallID:   tr.ToolCallID,
			ToolName:     tr.ToolName,
			Success:      tr.Success,
			ResultText:   tr.ResultText,
			ErrorMessage: tr.ErrorMessage,
		})
	}
}

func (h *Handler) handlePlanUpdate(sessionID string, data json.RawMessage) {
	var plan struct {
		Steps []struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		h.logger.Warn("failed to parse plan update", "error", err)
		return
	}
	if h.callbacks.OnPlanUpdate != nil {
		steps := make([]PlanStep, len(plan.Steps))
		for i, s := range plan.Steps {
			steps[i] = PlanStep{Title: s.Title, Status: s.Status}
		}
		h.callbacks.OnPlanUpdate(sessionID, PlanUpdate{Steps: steps})
	}
}

func (h *Handler) handleThinkingUpdate(sessionID string, data json.RawMessage) {
	var t struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		h.logger.Warn("failed to parse thinking update", "error", err)
		return
	}
	if h.callbacks.OnThinkingUpdate != nil {
		h.callbacks.OnThinkingUpdate(sessionID, ThinkingUpdate{Text: t.Text})
	}
}

func (h *Handler) handleSessionComplete(_ json.RawMessage) {
	if h.callbacks.OnStateChange != nil {
		h.callbacks.OnStateChange(StateIdle)
	}
}

func (h *Handler) handlePermissionRequest(params json.RawMessage) {
	var req struct {
		SessionID     string `json:"session_id"`
		RequestID     string `json:"request_id"`
		ToolName      string `json:"tool_name"`
		ArgumentsJSON string `json:"arguments_json"`
		Description   string `json:"description"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Warn("failed to parse permission/request", "error", err)
		return
	}

	if h.callbacks.OnStateChange != nil {
		h.callbacks.OnStateChange(StateWaitingPermission)
	}
	if h.callbacks.OnPermissionRequest != nil {
		h.callbacks.OnPermissionRequest(PermissionRequest{
			SessionID:     req.SessionID,
			RequestID:     req.RequestID,
			ToolName:      req.ToolName,
			ArgumentsJSON: req.ArgumentsJSON,
			Description:   req.Description,
		})
	}
}
