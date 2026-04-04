package acp

import (
	"encoding/json"
	"fmt"
)

// mockHandlePermissionPrompt sends a permission request, waits for response,
// then sends content. Used by mockModeSendPerm.
func mockHandlePermissionPrompt(id int64, reader *Reader, writer *Writer) {
	permID, _ := writer.WriteRequest("session/request_permission", map[string]any{
		"sessionId": "mock-session-001",
		"toolCall":  map[string]any{"toolCallId": "tc-perm-1", "title": "exec_command"},
		"options": []map[string]any{
			{"optionId": "opt-allow", "name": "Allow once", "kind": "allow_once"},
			{"optionId": "opt-reject", "name": "Reject once", "kind": "reject_once"},
		},
	})
	// Wait for permission response from client.
	for {
		resp, err := reader.ReadMessage()
		if err != nil {
			return
		}
		if resp.IsResponse() {
			respID, _ := resp.GetID()
			if respID == permID {
				break
			}
		}
	}
	mockSendDefaultContent(id, writer)
}

// mockHandleValidateInit validates the initialize params format.
func mockHandleValidateInit(id int64, params []byte, writer *Writer) {
	var p map[string]any
	if err := json.Unmarshal(params, &p); err != nil {
		writeValidationError(id, writer, "initialize: invalid JSON params")
		return
	}
	// protocolVersion must be a number == 1
	pv, ok := p["protocolVersion"]
	if !ok {
		writeValidationError(id, writer, "initialize: missing protocolVersion")
		return
	}
	if pvNum, ok := pv.(float64); !ok || pvNum != 1 {
		writeValidationError(id, writer, "initialize: protocolVersion must be 1")
		return
	}
	// clientInfo must be an object with "name"
	ci, ok := p["clientInfo"]
	if !ok {
		writeValidationError(id, writer, "initialize: missing clientInfo")
		return
	}
	ciMap, ok := ci.(map[string]any)
	if !ok {
		writeValidationError(id, writer, "initialize: clientInfo must be object")
		return
	}
	if _, ok := ciMap["name"]; !ok {
		writeValidationError(id, writer, "initialize: clientInfo.name missing")
		return
	}
	// clientCapabilities must be an object
	cc, ok := p["clientCapabilities"]
	if !ok {
		writeValidationError(id, writer, "initialize: missing clientCapabilities")
		return
	}
	if _, ok := cc.(map[string]any); !ok {
		writeValidationError(id, writer, "initialize: clientCapabilities must be object")
		return
	}
	writer.WriteResponse(id, map[string]any{
		"protocol_version": "2025-01-01",
		"capabilities":     map[string]any{"permissions": true},
	}, nil)
}

// mockHandleValidateSessionNew validates the session/new params format.
func mockHandleValidateSessionNew(id int64, params []byte, writer *Writer) {
	var p map[string]any
	if err := json.Unmarshal(params, &p); err != nil {
		writeValidationError(id, writer, "session/new: invalid JSON params")
		return
	}
	cwd, ok := p["cwd"]
	if !ok {
		writeValidationError(id, writer, "session/new: missing cwd")
		return
	}
	cwdStr, ok := cwd.(string)
	if !ok || cwdStr == "" {
		writeValidationError(id, writer, "session/new: cwd must be non-empty string")
		return
	}
	servers, ok := p["mcpServers"]
	if !ok {
		writeValidationError(id, writer, "session/new: missing mcpServers")
		return
	}
	if _, ok := servers.([]any); !ok {
		writeValidationError(id, writer, "session/new: mcpServers must be array")
		return
	}
	writer.WriteResponse(id, map[string]any{"sessionId": "mock-session-001"}, nil)
}

// mockHandleValidatePrompt validates the session/prompt params format.
func mockHandleValidatePrompt(id int64, params []byte, writer *Writer) {
	var p map[string]any
	if err := json.Unmarshal(params, &p); err != nil {
		writeValidationError(id, writer, "session/prompt: invalid JSON params")
		return
	}
	if _, ok := p["sessionId"]; !ok {
		writeValidationError(id, writer, "session/prompt: missing sessionId")
		return
	}
	prompt, ok := p["prompt"]
	if !ok {
		writeValidationError(id, writer, "session/prompt: missing prompt")
		return
	}
	arr, ok := prompt.([]any)
	if !ok {
		writeValidationError(id, writer, "session/prompt: prompt must be array")
		return
	}
	for i, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			writeValidationError(id, writer, fmt.Sprintf("session/prompt: prompt[%d] must be object", i))
			return
		}
		if _, ok := obj["type"]; !ok {
			writeValidationError(id, writer, fmt.Sprintf("session/prompt: prompt[%d] missing type", i))
			return
		}
	}
	mockSendDefaultContent(id, writer)
}

// writeValidationError sends a JSON-RPC error response with the given message.
func writeValidationError(id int64, writer *Writer, msg string) {
	writer.WriteResponse(id, nil, &JSONRPCError{
		Code: ErrCodeInvalidParams, Message: msg,
	})
}
