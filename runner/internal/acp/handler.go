package acp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// permissionOption mirrors the ACP PermissionOption for internal storage.
type permissionOption struct {
	OptionID string `json:"optionId"`
	Kind     string `json:"kind"` // allow_once, allow_always, reject_once, reject_always
}

// Handler dispatches inbound JSON-RPC notifications from the agent.
// Designed for the standard ACP protocol (Gemini CLI, OpenCode).
type Handler struct {
	callbacks EventCallbacks
	logger    *slog.Logger

	// pendingOptions stores permission options keyed by requestID
	// so RespondToPermission can select the correct optionId.
	pendingOptions   map[string][]permissionOption
	pendingOptionsMu sync.Mutex
}

// NewHandler creates a Handler that routes notifications to the
// provided callbacks.
func NewHandler(callbacks EventCallbacks, logger *slog.Logger) *Handler {
	return &Handler{
		callbacks:      callbacks,
		logger:         logger,
		pendingOptions: make(map[string][]permissionOption),
	}
}

// storePermissionOptions saves the agent-provided options for later use.
func (h *Handler) storePermissionOptions(requestID string, options []struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}) {
	h.pendingOptionsMu.Lock()
	defer h.pendingOptionsMu.Unlock()
	opts := make([]permissionOption, len(options))
	for i, o := range options {
		opts[i] = permissionOption{OptionID: o.OptionID, Kind: o.Kind}
	}
	h.pendingOptions[requestID] = opts
}

// SelectOptionID picks the appropriate optionId for the given requestID.
// Returns the first allow_once/allow_always option if approved, or
// reject_once/reject_always if denied. Returns "" if no matching option.
func (h *Handler) SelectOptionID(requestID string, approved bool) string {
	h.pendingOptionsMu.Lock()
	opts := h.pendingOptions[requestID]
	delete(h.pendingOptions, requestID)
	h.pendingOptionsMu.Unlock()

	for _, o := range opts {
		if approved && (o.Kind == "allow_once" || o.Kind == "allow_always") {
			return o.OptionID
		}
		if !approved && (o.Kind == "reject_once" || o.Kind == "reject_always") {
			return o.OptionID
		}
	}
	// Fallback: return first option or empty
	if len(opts) > 0 {
		return opts[0].OptionID
	}
	return ""
}

// HandleNotification processes an inbound notification from the agent.
func (h *Handler) HandleNotification(method string, params json.RawMessage) {
	switch {
	case method == "session/update":
		h.handleSessionUpdate(params)
	case strings.HasPrefix(method, "_loopal/"):
		h.handleLoopalExt(method, params)
	default:
		h.logger.Debug("unhandled notification", "method", method)
	}
}

// handleLoopalExt forwards Loopal control-panel `_loopal/*` extension
// notifications (bg shell / cron / task / topology) to OnLoopalExt. The
// `_loopal/` prefix is stripped so downstream relay event types read
// `loopal.<kind>`.
// loopalPanelKinds are the only _loopal/* extensions the runner forwards as
// control-panel events. Session-level extensions (permission_resolved,
// question_resolved, retryError, tokenUsage, inbox.*, cleared, ...) are dropped
// — they carry no `data` field (which would crash sendAcpViaRelay's flatten on
// a nil map) and the GUI's Loopal store has no consumer for them.
var loopalPanelKinds = map[string]bool{
	"bgTask.spawned": true, "bgTask.output": true, "bgTask.completed": true,
	"crons": true, "tasks": true, "mcp": true, "topology.spawn": true, "goal": true,
	"mode": true, "thinking": true, "model": true,
}

func (h *Handler) handleLoopalExt(method string, params json.RawMessage) {
	kind := strings.TrimPrefix(method, "_loopal/")
	// permission_mode is an ACP configuration concept, not a control-panel
	// signal — route it to OnConfigChange so it lands in the session's
	// Configuration (the permission selector's source), not loopalState.
	if kind == "permission_mode" {
		h.handleLoopalPermissionMode(params)
		return
	}
	if h.callbacks.OnLoopalExt == nil {
		return
	}
	if !loopalPanelKinds[kind] {
		return
	}
	var p struct {
		SessionID string          `json:"sessionId"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		h.logger.Warn("failed to parse _loopal notification", "method", method, "error", err)
		return
	}
	h.callbacks.OnLoopalExt(p.SessionID, kind, p.Data)
}

// handleLoopalPermissionMode routes _loopal/permission_mode to OnConfigChange so
// the agent's current permission mode lands in the session Configuration (where
// the permission selector reads it). loopal emits this on cold-start + switch.
func (h *Handler) handleLoopalPermissionMode(params json.RawMessage) {
	if h.callbacks.OnConfigChange == nil {
		return
	}
	var p struct {
		SessionID string `json:"sessionId"`
		Data      struct {
			PermissionMode string `json:"permission_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		h.logger.Warn("failed to parse _loopal/permission_mode", "error", err)
		return
	}
	if p.Data.PermissionMode != "" {
		h.callbacks.OnConfigChange(p.SessionID, ConfigUpdate{PermissionMode: p.Data.PermissionMode})
	}
}

// HandlePermissionRequest processes a session/request_permission JSON-RPC request.
// The JSON-RPC request id is passed so RespondToPermission can send a proper response.
func (h *Handler) HandlePermissionRequest(rpcID int64, params json.RawMessage) {
	var req struct {
		SessionID string `json:"sessionId"`
		ToolCall  struct {
			ToolCallID string `json:"toolCallId"`
			Title      string `json:"title"`
		} `json:"toolCall"`
		Options []struct {
			OptionID string `json:"optionId"`
			Name     string `json:"name"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		h.logger.Warn("failed to parse session/request_permission", "error", err)
		return
	}

	if h.callbacks.OnStateChange != nil {
		h.callbacks.OnStateChange(StateWaitingPermission)
	}
	if h.callbacks.OnPermissionRequest != nil {
		toolName := req.ToolCall.Title
		if toolName == "" {
			toolName = req.ToolCall.ToolCallID
		}
		// Store options so RespondToPermission can select the correct optionId.
		h.storePermissionOptions(fmt.Sprintf("%d", rpcID), req.Options)
		h.callbacks.OnPermissionRequest(PermissionRequest{
			SessionID:   req.SessionID,
			RequestID:   fmt.Sprintf("%d", rpcID),
			ToolName:    toolName,
			Description: fmt.Sprintf("Tool: %s", req.ToolCall.ToolCallID),
		})
	}
}

// handleSessionUpdate parses the standard ACP session/update notification:
// {"sessionId":"...","update":{"sessionUpdate":"agent_message_chunk",...}}
func (h *Handler) handleSessionUpdate(params json.RawMessage) {
	var raw struct {
		SessionID string          `json:"sessionId"`
		Update    json.RawMessage `json:"update"`
	}
	if err := json.Unmarshal(params, &raw); err != nil {
		h.logger.Warn("failed to parse session/update", "error", err)
		return
	}

	// Extract the discriminator field.
	var disc struct {
		SessionUpdate string `json:"sessionUpdate"`
	}
	if err := json.Unmarshal(raw.Update, &disc); err != nil {
		h.logger.Warn("failed to parse session/update discriminator", "error", err)
		return
	}

	switch disc.SessionUpdate {
	case "agent_message_chunk":
		h.handleMessageChunk(raw.SessionID, "assistant", raw.Update)
	case "user_message_chunk":
		h.handleMessageChunk(raw.SessionID, "user", raw.Update)
	case "agent_thought_chunk":
		h.handleThoughtChunk(raw.SessionID, raw.Update)
	case "tool_call":
		h.handleToolCall(raw.SessionID, raw.Update)
	case "tool_call_update":
		h.handleToolCallUpdate(raw.SessionID, raw.Update)
	case "plan":
		h.handlePlanUpdate(raw.SessionID, raw.Update)
	default:
		h.logger.Debug("unhandled session/update type", "sessionUpdate", disc.SessionUpdate)
	}
}

// handleMessageChunk extracts text from a ContentBlock and fires OnContentChunk.
func (h *Handler) handleMessageChunk(sessionID, role string, data json.RawMessage) {
	var msg struct {
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Warn("failed to parse message chunk", "error", err)
		return
	}
	if h.callbacks.OnContentChunk != nil {
		h.callbacks.OnContentChunk(sessionID, ContentChunk{
			Text: msg.Content.Text, Role: role,
		})
	}
}

// handleThoughtChunk extracts text from a ContentBlock and fires OnThinkingUpdate.
func (h *Handler) handleThoughtChunk(sessionID string, data json.RawMessage) {
	var msg struct {
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Warn("failed to parse thought chunk", "error", err)
		return
	}
	if h.callbacks.OnThinkingUpdate != nil {
		h.callbacks.OnThinkingUpdate(sessionID, ThinkingUpdate{Text: msg.Content.Text})
	}
}

// handleToolCall handles the initial tool_call update (status: pending/in_progress).
func (h *Handler) handleToolCall(sessionID string, data json.RawMessage) {
	var tc struct {
		ToolCallID string `json:"toolCallId"`
		Title      string `json:"title"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(data, &tc); err != nil {
		h.logger.Warn("failed to parse tool_call", "error", err)
		return
	}
	status := tc.Status
	if status == "pending" || status == "" {
		status = "running"
	}
	if h.callbacks.OnToolCallUpdate != nil {
		h.callbacks.OnToolCallUpdate(sessionID, ToolCallUpdate{
			ToolCallID: tc.ToolCallID,
			ToolName:   tc.Title,
			Status:     status,
		})
	}
}

// handleToolCallUpdate handles tool_call_update (status changes, results).
func (h *Handler) handleToolCallUpdate(sessionID string, data json.RawMessage) {
	var tc struct {
		ToolCallID   string `json:"toolCallId"`
		Title        string `json:"title"`
		Status       string `json:"status"`
		ResultText   string `json:"resultText"`
		ErrorMessage string `json:"errorMessage"`
	}
	if err := json.Unmarshal(data, &tc); err != nil {
		h.logger.Warn("failed to parse tool_call_update", "error", err)
		return
	}
	if h.callbacks.OnToolCallUpdate != nil {
		h.callbacks.OnToolCallUpdate(sessionID, ToolCallUpdate{
			ToolCallID: tc.ToolCallID,
			ToolName:   tc.Title,
			Status:     tc.Status,
		})
	}
	// If status is terminal, also fire a result callback.
	if tc.Status == "completed" || tc.Status == "failed" {
		if h.callbacks.OnToolCallResult != nil {
			h.callbacks.OnToolCallResult(sessionID, ToolCallResult{
				ToolCallID:   tc.ToolCallID,
				ToolName:     tc.Title,
				Success:      tc.Status == "completed",
				ResultText:   tc.ResultText,
				ErrorMessage: tc.ErrorMessage,
			})
		}
	}
}

// handlePlanUpdate handles plan notifications (entries → steps mapping).
func (h *Handler) handlePlanUpdate(sessionID string, data json.RawMessage) {
	var plan struct {
		Entries []struct {
			Content  string `json:"content"`
			Priority string `json:"priority"` // high, medium, low
			Status   string `json:"status"`   // pending, in_progress, completed
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &plan); err != nil {
		h.logger.Warn("failed to parse plan update", "error", err)
		return
	}
	if h.callbacks.OnPlanUpdate != nil {
		steps := make([]PlanStep, len(plan.Entries))
		for i, e := range plan.Entries {
			steps[i] = PlanStep{Title: e.Content, Status: e.Status}
		}
		h.callbacks.OnPlanUpdate(sessionID, PlanUpdate{Steps: steps})
	}
}
