package claude

import (
	"encoding/json"
)

// --- Claude stream-json NDJSON message types ---

// message is the top-level message read from Claude's stdout.
type message struct {
	Type    string          `json:"type"`    // system, stream_event, assistant, user, result, control_request, control_response
	Subtype string          `json:"subtype"` // init, api_retry, success, error_*
	UUID    string          `json:"uuid"`
	Event   json.RawMessage `json:"event"`   // stream_event only
	Message json.RawMessage `json:"message"` // assistant/user only
	Result  string          `json:"result"`  // result only
	IsError bool            `json:"is_error"`

	// Fields present on system/init:
	SessionID string          `json:"session_id"`
	Tools     json.RawMessage `json:"tools"`       // []string or int (version-dependent)
	MCP       json.RawMessage `json:"mcp_servers"` // []object or int (version-dependent)

	// Fields present on result:
	NumTurns int     `json:"num_turns"`
	Duration float64 `json:"duration_ms"`

	// Fields present on control_request / control_response:
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response"` // control_response only
}

// controlInitMessage is written to stdin to trigger the initialize handshake.
type controlInitMessage struct {
	Type      string             `json:"type"` // "control_request"
	RequestID string             `json:"request_id"`
	Request   controlInitPayload `json:"request"`
}

// controlInitPayload is the request body for the initialize control_request.
type controlInitPayload struct {
	Subtype string `json:"subtype"` // "initialize"
}

// streamEvent is the inner event for type=stream_event messages.
type streamEvent struct {
	Type         string          `json:"type"` // message_start, content_block_start/delta/stop, message_delta/stop
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block"` // content_block_start
	Delta        json.RawMessage `json:"delta"`         // content_block_delta, message_delta
}

// contentBlock is a content block in a Claude message.
type contentBlock struct {
	Type  string `json:"type"`  // text, tool_use, tool_result, thinking
	ID    string `json:"id"`    // tool_use only
	Name  string `json:"name"`  // tool_use only
	Text  string `json:"text"`  // text, thinking
	Input any    `json:"input"` // tool_use (complete)
}

// delta is a streaming delta for content_block_delta.
type delta struct {
	Type        string `json:"type"`         // text_delta, input_json_delta, thinking_delta
	Text        string `json:"text"`         // text_delta, thinking_delta
	PartialJSON string `json:"partial_json"` // input_json_delta
}

// userMessage is the complete user message (type=user).
type userMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // can be string or array
}

// userInput is written to stdin to send a prompt.
type userInput struct {
	Type            string       `json:"type"` // "user"
	SessionID       string       `json:"session_id,omitempty"`
	ParentToolUseID *string      `json:"parent_tool_use_id,omitempty"`
	Message         userInputMsg `json:"message"`
}

// userInputMsg is the message body of a stdin user input.
type userInputMsg struct {
	Role    string `json:"role"` // "user"
	Content string `json:"content"`
}

// toolCallState tracks in-progress tool_use assembly during streaming.
type toolCallState struct {
	ID        string
	Name      string
	InputJSON string // accumulated JSON fragments
}

// --- Control protocol types (permission flow) ---

// controlRequestPayload is the "request" field of a control_request message.
type controlRequestPayload struct {
	Subtype     string          `json:"subtype"` // "can_use_tool"
	ToolName    string          `json:"tool_name"`
	ToolUseID   string          `json:"tool_use_id"`
	Input       json.RawMessage `json:"input"`
	Description string          `json:"description"`
}

// controlResponseMessage is written to stdin to respond to a control_request.
type controlResponseMessage struct {
	Type     string                 `json:"type"` // "control_response"
	Response controlResponsePayload `json:"response"`
}

// controlResponsePayload is the "response" field of a control_response.
type controlResponsePayload struct {
	Subtype   string              `json:"subtype"` // "success"
	RequestID string              `json:"request_id"`
	Response  controlResponseData `json:"response"`
}

// controlResponseData holds the permission decision.
type controlResponseData struct {
	Behavior     string          `json:"behavior"`               // "allow" | "deny"
	UpdatedInput json.RawMessage `json:"updatedInput,omitempty"` // allow: pass original input
	Message      string          `json:"message,omitempty"`      // deny: reason message
}
