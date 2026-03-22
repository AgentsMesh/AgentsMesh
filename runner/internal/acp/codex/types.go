package codex

// --- Codex app-server JSON-RPC types ---

// threadStartResult is the response from thread/start.
type threadStartResult struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

// turnStartParams are the parameters for turn/start.
type turnStartParams struct {
	ThreadID string    `json:"thread_id"`
	Input    turnInput `json:"input"`
}

// turnInput is the input for a turn.
type turnInput struct {
	Type    string `json:"type"`    // "text"
	Content string `json:"content"`
}

// turnInterruptParams are the parameters for turn/interrupt.
type turnInterruptParams struct {
	ThreadID string `json:"thread_id"`
}

// approvalRequest is an incoming approval request from the Codex agent.
type approvalRequest struct {
	RequestID   string `json:"request_id"`
	Type        string `json:"type"`        // "command_execution", "patch_apply"
	Command     string `json:"command"`
	Description string `json:"description"`
}

// serverRequestResponseParams are sent to respond to a serverRequest.
type serverRequestResponseParams struct {
	RequestID string `json:"request_id"`
	Approved  bool   `json:"approved"`
}

// agentMessageDelta carries streaming text from the agent.
type agentMessageDelta struct {
	Text string `json:"text"`
}

// thinkingDelta carries streaming thinking text.
type thinkingDelta struct {
	Text string `json:"text"`
}

// planDelta carries a plan step update.
type planDelta struct {
	Step struct {
		Title  string `json:"title"`
		Status string `json:"status"` // "pending", "in_progress", "completed"
	} `json:"step"`
}

// toolCallStarted is sent when a tool call begins.
type toolCallStarted struct {
	ToolCallID    string `json:"tool_call_id"`
	ToolName      string `json:"tool_name"`
	ArgumentsJSON string `json:"arguments_json"`
}

// commandExecutionStarted is sent when a shell command starts.
type commandExecutionStarted struct {
	ToolCallID string `json:"tool_call_id"`
	Command    string `json:"command"`
}

// commandExecutionCompleted is sent when a shell command completes.
type commandExecutionCompleted struct {
	ToolCallID string `json:"tool_call_id"`
	ExitCode   int    `json:"exit_code"`
	Output     string `json:"output"`
}

// itemCompleted is sent when a streamed item (e.g., tool_call) finishes.
type itemCompleted struct {
	Type       string `json:"type"` // "tool_call", "agent_message", etc.
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
}
