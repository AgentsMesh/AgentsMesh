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
	ThreadID string      `json:"threadId"`
	Input    []turnInput `json:"input"`
}

// turnInput is an input item for a turn.
type turnInput struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// turnInterruptParams are the parameters for turn/interrupt.
type turnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId,omitempty"`
}

// approvalRequestParams is an incoming approval request from the Codex agent
// (received as a JSON-RPC request, not notification).
type approvalRequestParams struct {
	Command     string `json:"command,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
}

// agentMessageDelta carries streaming text from the agent.
type agentMessageDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

// reasoningDelta carries streaming reasoning/thinking text.
type reasoningDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

// planDelta carries a plan text stream update.
type planDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

// itemStartedParams is the generic item/started notification.
// The nested item.type distinguishes commandExecution, toolCall, fileChange, etc.
type itemStartedParams struct {
	Item struct {
		ID      string `json:"id"`
		Type    string `json:"type"` // "commandExecution", "toolCall", "fileChange", etc.
		Command []struct {
			Value string `json:"value"`
		} `json:"command,omitempty"` // commandExecution only
		ToolName string `json:"toolName,omitempty"` // toolCall only
		FilePath string `json:"filePath,omitempty"` // fileChange only
	} `json:"item"`
}

// itemCompletedParams is the generic item/completed notification.
// Includes type-specific fields for commandExecution (exitCode, aggregatedOutput).
type itemCompletedParams struct {
	Item struct {
		ID               string `json:"id"`
		Type             string `json:"type"`
		Status           string `json:"status,omitempty"`
		ExitCode         *int   `json:"exitCode,omitempty"`         // commandExecution
		AggregatedOutput string `json:"aggregatedOutput,omitempty"` // commandExecution
		ToolName         string `json:"toolName,omitempty"`         // toolCall
		FilePath         string `json:"filePath,omitempty"`         // fileChange
	} `json:"item"`
}

// turnCompletedParams are the parameters for turn/completed notification.
// The status and error are nested inside a "turn" object per Codex protocol.
type turnCompletedParams struct {
	Turn struct {
		Status string `json:"status"` // "completed", "failed", "interrupted"
		Error  *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"turn"`
}
