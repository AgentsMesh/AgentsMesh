package acp

import (
	"fmt"
)

// NewSession creates a new ACP session with optional MCP servers config.
func (c *ACPClient) NewSession(mcpServers map[string]any) error {
	sessionID, err := c.transport.NewSession(c.cfg.WorkDir, mcpServers)
	if err != nil {
		return err
	}

	// Only update sessionID if the transport returned a non-empty one.
	// Claude transport returns "" here because its session_id was already
	// set during Handshake() via the system/init message.
	if sessionID != "" {
		c.sessionMu.Lock()
		c.sessionID = sessionID
		c.sessionMu.Unlock()
	}

	c.logger.Info("ACP session created", "session_id", c.SessionID())
	return nil
}

// SendPrompt sends a prompt to the current session.
func (c *ACPClient) SendPrompt(prompt string) error {
	if c.State() != StateIdle {
		return fmt.Errorf("cannot send prompt in state %s", c.State())
	}

	c.setState(StateProcessing)

	// Record user message in history so snapshots include it.
	c.addMessage(ContentChunk{Text: prompt, Role: "user"})

	if err := c.transport.SendPrompt(c.SessionID(), prompt); err != nil {
		c.setState(StateIdle)
		return fmt.Errorf("write prompt: %w", err)
	}

	return nil
}

// RespondToPermission approves or denies a permission request.
// updatedInput is optional; when non-nil, it replaces the tool's original input (for AskUserQuestion).
func (c *ACPClient) RespondToPermission(requestID string, approved bool, updatedInput map[string]any) error {
	if err := c.transport.RespondToPermission(requestID, approved, updatedInput); err != nil {
		return fmt.Errorf("write permission response: %w", err)
	}
	if c.State() == StateWaitingPermission {
		c.setState(StateProcessing)
	}
	return nil
}

// CancelSession cancels the current session's processing.
func (c *ACPClient) CancelSession() error {
	return c.transport.CancelSession(c.SessionID())
}

// Interrupt sends an interrupt control_request to stop the current processing.
func (c *ACPClient) Interrupt() error {
	_, err := c.transport.SendControlRequest(c.SessionID(), "interrupt", nil)
	return err
}

// SetPermissionMode dynamically changes the permission mode at runtime.
// On success, fires OnConfigChange so the wrapped state and downstream
// subscribers learn the new value via the same path as data-plane updates.
func (c *ACPClient) SetPermissionMode(mode string) error {
	_, err := c.transport.SendControlRequest(c.SessionID(), "set_permission_mode", map[string]any{
		"mode": mode,
	})
	if err != nil {
		return err
	}
	if c.cfg.Callbacks.OnConfigChange != nil {
		c.cfg.Callbacks.OnConfigChange(c.SessionID(), ConfigUpdate{PermissionMode: mode})
	}
	return nil
}

// SetModel dynamically changes the AI model at runtime.
// On success, fires OnConfigChange (see SetPermissionMode for rationale).
func (c *ACPClient) SetModel(model string) error {
	_, err := c.transport.SendControlRequest(c.SessionID(), "set_model", map[string]any{
		"model": model,
	})
	if err != nil {
		return err
	}
	if c.cfg.Callbacks.OnConfigChange != nil {
		c.cfg.Callbacks.OnConfigChange(c.SessionID(), ConfigUpdate{Model: model})
	}
	return nil
}

// GetContextUsage queries the current context window usage.
func (c *ACPClient) GetContextUsage() (map[string]any, error) {
	return c.transport.SendControlRequest(c.SessionID(), "get_context_usage", nil)
}

// SendControlRequest sends a generic outgoing control_request.
func (c *ACPClient) SendControlRequest(subtype string, payload map[string]any) (map[string]any, error) {
	return c.transport.SendControlRequest(c.SessionID(), subtype, payload)
}

// GetRecentMessages returns recent content messages formatted as text.
func (c *ACPClient) GetRecentMessages(n int) string {
	c.messagesMu.RLock()
	defer c.messagesMu.RUnlock()

	start := 0
	if n > 0 && n < len(c.messages) {
		start = len(c.messages) - n
	}

	var result string
	for _, msg := range c.messages[start:] {
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("[%s] %s", msg.Role, msg.Text)
	}
	return result
}

// AddPendingPermission records a permission request that is awaiting a response.
func (c *ACPClient) AddPendingPermission(req PermissionRequest) {
	c.pendingPermsMu.Lock()
	defer c.pendingPermsMu.Unlock()
	c.pendingPerms = append(c.pendingPerms, req)
}

// RemovePendingPermission removes a permission request by its request ID
// (called after the user approves or denies it).
func (c *ACPClient) RemovePendingPermission(requestID string) {
	c.pendingPermsMu.Lock()
	defer c.pendingPermsMu.Unlock()
	for i, p := range c.pendingPerms {
		if p.RequestID == requestID {
			c.pendingPerms = append(c.pendingPerms[:i], c.pendingPerms[i+1:]...)
			return
		}
	}
}

// GetSessionSnapshot returns a snapshot of the current ACP session state,
// including message history, tool calls, plan, and pending permissions.
func (c *ACPClient) GetSessionSnapshot() *AcpSessionSnapshot {
	c.messagesMu.RLock()
	msgs := make([]ContentChunk, len(c.messages))
	copy(msgs, c.messages)
	c.messagesMu.RUnlock()

	c.toolCallsMu.RLock()
	toolCalls := make([]ToolCallSnapshot, 0, len(c.toolCalls))
	for _, tc := range c.toolCalls {
		snap := *tc // copy
		toolCalls = append(toolCalls, snap)
	}
	c.toolCallsMu.RUnlock()

	c.planMu.RLock()
	plan := make([]PlanStep, len(c.plan))
	copy(plan, c.plan)
	c.planMu.RUnlock()

	c.pendingPermsMu.RLock()
	perms := make([]PermissionRequest, len(c.pendingPerms))
	copy(perms, c.pendingPerms)
	c.pendingPermsMu.RUnlock()

	c.thinkingsMu.RLock()
	thinkings := make([]ThinkingUpdate, len(c.thinkings))
	copy(thinkings, c.thinkings)
	c.thinkingsMu.RUnlock()

	c.logsMu.RLock()
	logs := make([]LogEntry, len(c.logs))
	copy(logs, c.logs)
	c.logsMu.RUnlock()

	return &AcpSessionSnapshot{
		SessionID:          c.SessionID(),
		State:              c.State(),
		Messages:           msgs,
		ToolCalls:          toolCalls,
		Plan:               plan,
		Thinkings:          thinkings,
		Logs:               logs,
		PendingPermissions: perms,
		Configuration:      c.Configuration(),
	}
}
