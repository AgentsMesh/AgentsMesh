package codex

type threadStartResult struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

type turnStartParams struct {
	ThreadID string      `json:"threadId"`
	Input    []turnInput `json:"input"`
}

type turnInput struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type turnInterruptParams struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId,omitempty"`
}

type approvalRequestParams struct {
	Command     string `json:"command,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
}

type agentMessageDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

type reasoningDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

type planDelta struct {
	ItemID string `json:"itemId"`
	Delta  string `json:"delta"`
}

type itemStartedParams struct {
	Item struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Command []struct {
			Value string `json:"value"`
		} `json:"command,omitempty"`
		ToolName string `json:"toolName,omitempty"`
		FilePath string `json:"filePath,omitempty"`
	} `json:"item"`
}

type itemCompletedParams struct {
	Item struct {
		ID               string `json:"id"`
		Type             string `json:"type"`
		Status           string `json:"status,omitempty"`
		ExitCode         *int   `json:"exitCode,omitempty"`
		AggregatedOutput string `json:"aggregatedOutput,omitempty"`
		ToolName         string `json:"toolName,omitempty"`
		FilePath         string `json:"filePath,omitempty"`
	} `json:"item"`
}

type turnCompletedParams struct {
	Turn struct {
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"turn"`
}
