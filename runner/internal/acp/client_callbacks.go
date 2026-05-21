package acp

// wrapCallbacks decorates the user-provided EventCallbacks with internal
// state-keeping (message/tool/plan/thinking/log/config accumulators) so that
// GetSessionSnapshot reflects the latest state. The wrapped callback chain
// also keeps the original user callback as a tail-call.
//
// OnStateChange is special: setState already fires the user's original
// callback, so the wrap MUST replace it (not chain) — otherwise late
// subscribers would observe duplicate state transitions.
func (c *ACPClient) wrapCallbacks() EventCallbacks {
	wrapped := c.cfg.Callbacks

	originalOnContent := wrapped.OnContentChunk
	wrapped.OnContentChunk = func(sessionID string, chunk ContentChunk) {
		c.addMessage(chunk)
		if originalOnContent != nil {
			originalOnContent(sessionID, chunk)
		}
	}

	originalOnToolCallUpdate := wrapped.OnToolCallUpdate
	wrapped.OnToolCallUpdate = func(sessionID string, update ToolCallUpdate) {
		c.upsertToolCall(update)
		if originalOnToolCallUpdate != nil {
			originalOnToolCallUpdate(sessionID, update)
		}
	}

	originalOnToolCallResult := wrapped.OnToolCallResult
	wrapped.OnToolCallResult = func(sessionID string, result ToolCallResult) {
		c.applyToolCallResult(result)
		if originalOnToolCallResult != nil {
			originalOnToolCallResult(sessionID, result)
		}
	}

	originalOnPlanUpdate := wrapped.OnPlanUpdate
	wrapped.OnPlanUpdate = func(sessionID string, update PlanUpdate) {
		c.setPlan(update.Steps)
		if originalOnPlanUpdate != nil {
			originalOnPlanUpdate(sessionID, update)
		}
	}

	originalOnThinking := wrapped.OnThinkingUpdate
	wrapped.OnThinkingUpdate = func(sessionID string, update ThinkingUpdate) {
		c.addThinking(update)
		if originalOnThinking != nil {
			originalOnThinking(sessionID, update)
		}
	}

	originalOnLog := wrapped.OnLog
	wrapped.OnLog = func(level, message string) {
		c.addLog(LogEntry{Level: level, Message: message})
		if originalOnLog != nil {
			originalOnLog(level, message)
		}
	}

	originalOnConfigChange := wrapped.OnConfigChange
	wrapped.OnConfigChange = func(sessionID string, update ConfigUpdate) {
		merged := c.applyConfiguration(update)
		if originalOnConfigChange != nil {
			originalOnConfigChange(sessionID, ConfigUpdate(merged))
		}
	}

	wrapped.OnStateChange = func(newState string) {
		c.setState(newState)
	}

	return wrapped
}
