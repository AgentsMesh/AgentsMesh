package acp

// mockHandleMultiUpdatePrompt sends all 6 session/update types before responding.
// Used by mockModeMultiUpdate to test that the client correctly handles every
// update variant in a single prompt turn.
func mockHandleMultiUpdatePrompt(id int64, writer *Writer) {
	sid := "mock-session-001"

	// 1. agent_message_chunk
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "thinking..."},
		},
	})

	// 2. agent_thought_chunk
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]any{"type": "text", "text": "let me think"},
		},
	})

	// 3. tool_call (pending)
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "tc-001",
			"title":         "read_file",
			"status":        "pending",
		},
	})

	// 4. tool_call_update (completed)
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "tool_call_update",
			"toolCallId":    "tc-001",
			"status":        "completed",
		},
	})

	// 5. plan
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "plan",
			"entries": []map[string]any{
				{"content": "Step 1", "priority": "high", "status": "completed"},
				{"content": "Step 2", "priority": "medium", "status": "in_progress"},
			},
		},
	})

	// 6. user_message_chunk
	writer.WriteNotification("session/update", map[string]any{
		"sessionId": sid,
		"update": map[string]any{
			"sessionUpdate": "user_message_chunk",
			"content":       map[string]any{"type": "text", "text": "echoed user message"},
		},
	})

	writer.WriteResponse(id, map[string]any{"stopReason": "end_turn"}, nil)
}
