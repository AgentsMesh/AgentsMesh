package claude

import (
	"encoding/json"
)

type message struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype"`
	UUID    string          `json:"uuid"`
	Event   json.RawMessage `json:"event"`
	Message json.RawMessage `json:"message"`
	Result  string          `json:"result"`
	IsError bool            `json:"is_error"`

	SessionID string          `json:"session_id"`
	Tools     json.RawMessage `json:"tools"`
	MCP       json.RawMessage `json:"mcp_servers"`

	NumTurns int     `json:"num_turns"`
	Duration float64 `json:"duration_ms"`

	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response"`
}

type controlInitMessage struct {
	Type      string             `json:"type"`
	RequestID string             `json:"request_id"`
	Request   controlInitPayload `json:"request"`
}

type controlInitPayload struct {
	Subtype string `json:"subtype"`
}

type streamEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block"`
	Delta        json.RawMessage `json:"delta"`
}

type contentBlock struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Text  string `json:"text"`
	Input any    `json:"input"`
}

type delta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
}

type userMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type userInput struct {
	Type            string       `json:"type"`
	SessionID       string       `json:"session_id,omitempty"`
	ParentToolUseID *string      `json:"parent_tool_use_id,omitempty"`
	Message         userInputMsg `json:"message"`
}

type userInputMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolCallState struct {
	ID        string
	Name      string
	InputJSON string
}

type controlRequestPayload struct {
	Subtype     string          `json:"subtype"`
	ToolName    string          `json:"tool_name"`
	ToolUseID   string          `json:"tool_use_id"`
	Input       json.RawMessage `json:"input"`
	Description string          `json:"description"`
}

type controlResponseMessage struct {
	Type     string                 `json:"type"`
	Response controlResponsePayload `json:"response"`
}

type controlResponsePayload struct {
	Subtype   string              `json:"subtype"`
	RequestID string              `json:"request_id"`
	Response  controlResponseData `json:"response"`
}

type controlResponseData struct {
	Behavior           string          `json:"behavior"`
	UpdatedInput       json.RawMessage `json:"updatedInput,omitempty"`
	UpdatedPermissions json.RawMessage `json:"updatedPermissions,omitempty"`
	Message            string          `json:"message,omitempty"`
}
