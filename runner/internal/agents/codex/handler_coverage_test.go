package codex

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/stretchr/testify/assert"
)

func TestHandler_ItemStarted_FileChange(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/started", itemStartedParams{
		Item: struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Command []struct {
				Value string `json:"value"`
			} `json:"command,omitempty"`
			ToolName string `json:"toolName,omitempty"`
			FilePath string `json:"filePath,omitempty"`
		}{ID: "f1", Type: "fileChange", FilePath: "main.go"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.ToolUpdates, 1) {
		assert.Equal(t, "fileChange", f.ToolUpdates[0].ToolName)
		assert.Equal(t, "running", f.ToolUpdates[0].Status)
	}
}

func TestHandler_ItemCompleted_FileChange(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/completed", map[string]any{
		"item": map[string]any{
			"id": "f1", "type": "fileChange", "status": "completed", "filePath": "main.go",
		},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.ToolResults, 1) {
		assert.True(t, f.ToolResults[0].Success)
		assert.Equal(t, "main.go", f.ToolResults[0].ResultText)
	}
}

func TestHandler_ItemCompleted_CommandExecution_WithExitCode(t *testing.T) {
	f := newFixture()
	defer f.Close()
	exitCode := 1
	writeNotification(f.PW, "item/completed", itemCompletedParams{
		Item: struct {
			ID               string `json:"id"`
			Type             string `json:"type"`
			Status           string `json:"status,omitempty"`
			ExitCode         *int   `json:"exitCode,omitempty"`
			AggregatedOutput string `json:"aggregatedOutput,omitempty"`
			ToolName         string `json:"toolName,omitempty"`
			FilePath         string `json:"filePath,omitempty"`
		}{ID: "c1", Type: "commandExecution", ExitCode: &exitCode, AggregatedOutput: "error output"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.ToolResults, 1) {
		assert.False(t, f.ToolResults[0].Success)
		assert.Equal(t, "error output", f.ToolResults[0].ResultText)
	}
}

func TestHandler_TurnCompleted_Failed_WithError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "turn/completed", map[string]any{
		"turn": map[string]any{
			"status": "failed",
			"error":  map[string]any{"message": "something broke"},
		},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Contains(t, f.StateChanges, acp.StateIdle)
	if assert.Len(t, f.LogMessages, 1) {
		assert.Contains(t, f.LogMessages[0], "something broke")
	}
}

func TestHandler_TurnCompleted_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "turn/completed", "invalid")
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Contains(t, f.StateChanges, acp.StateIdle)
}

func TestHandler_ApprovalRequest_PathOnly(t *testing.T) {
	f := newFixture()
	defer f.Close()
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "item/fileChange/requestApproval",
		"params":  map[string]any{"path": "/tmp/file.go"},
	}
	data, _ := json.Marshal(msg)
	f.PW.Write(append(data, '\n'))
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.PermissionReqs, 1) {
		assert.Equal(t, "fileChange", f.PermissionReqs[0].ToolName)
		assert.Equal(t, "/tmp/file.go", f.PermissionReqs[0].Description)
	}
}

func TestHandler_ApprovalRequest_CommandOnly(t *testing.T) {
	f := newFixture()
	defer f.Close()
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      43,
		"method":  "item/commandExecution/requestApproval",
		"params":  map[string]any{"command": "rm -rf /"},
	}
	data, _ := json.Marshal(msg)
	f.PW.Write(append(data, '\n'))
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if assert.Len(t, f.PermissionReqs, 1) {
		assert.Equal(t, "command", f.PermissionReqs[0].ToolName)
		assert.Equal(t, "rm -rf /", f.PermissionReqs[0].Description)
	}
}

func TestHandler_ItemStarted_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/started", "invalid")
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Empty(t, f.ToolUpdates)
}

func TestHandler_ItemCompleted_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeNotification(f.PW, "item/completed", "invalid")
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Empty(t, f.ToolResults)
}

func TestHandler_NilCallbacks(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.handleNotification("item/agentMessage/delta", mustMarshal(agentMessageDelta{Delta: "x"}))
	tr.handleNotification("turn/completed", mustMarshal(map[string]any{"turn": map[string]any{"status": "completed"}}))
}

func TestTransport_Close_NoOp(t *testing.T) {
	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	tr.Close()
}

func mustMarshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
