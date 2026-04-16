package claude

import (
	"bufio"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_SendControlRequest_Interrupt(t *testing.T) {
	f := newFixtureWithStdin()
	defer f.Close()

	writeLine(f.PW, map[string]any{
		"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": "init_1"},
	})
	f.Drain()

	go func() {
		scanner := bufio.NewScanner(f.StdinPR)
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg["type"] == "control_request" {
				req := msg["request"].(map[string]any)
				if req["subtype"] == "interrupt" {
					writeLine(f.PW, map[string]any{
						"type": "control_response",
						"response": map[string]any{
							"subtype":    "success",
							"request_id": msg["request_id"],
							"response":   map[string]any{"interrupted": true},
						},
					})
					return
				}
			}
		}
	}()

	resp, err := f.transport.SendControlRequest("", "interrupt", nil)
	require.NoError(t, err)
	assert.Equal(t, true, resp["interrupted"])
}

func TestTransport_SendControlRequest_SetPermissionMode(t *testing.T) {
	f := newFixtureWithStdin()
	defer f.Close()

	writeLine(f.PW, map[string]any{
		"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": "init_1"},
	})
	f.Drain()

	go func() {
		scanner := bufio.NewScanner(f.StdinPR)
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg["type"] == "control_request" {
				req := msg["request"].(map[string]any)
				if req["subtype"] == "set_permission_mode" && req["mode"] == "acceptEdits" {
					writeLine(f.PW, map[string]any{
						"type": "control_response",
						"response": map[string]any{
							"subtype":    "success",
							"request_id": msg["request_id"],
							"response":   map[string]any{"mode": "acceptEdits"},
						},
					})
					return
				}
			}
		}
	}()

	resp, err := f.transport.SendControlRequest("", "set_permission_mode", map[string]any{"mode": "acceptEdits"})
	require.NoError(t, err)
	assert.Equal(t, "acceptEdits", resp["mode"])
}

func TestTransport_SendControlRequest_ErrorResponse(t *testing.T) {
	f := newFixtureWithStdin()
	defer f.Close()

	writeLine(f.PW, map[string]any{
		"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": "init_1"},
	})
	f.Drain()

	go func() {
		scanner := bufio.NewScanner(f.StdinPR)
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg["type"] == "control_request" {
				writeLine(f.PW, map[string]any{
					"type": "control_response",
					"response": map[string]any{
						"subtype":    "error",
						"request_id": msg["request_id"],
						"error":      "model not available",
					},
				})
				return
			}
		}
	}()

	_, err := f.transport.SendControlRequest("", "set_model", map[string]any{"model": "invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model not available")
}

func TestTransport_SendControlRequest_ConcurrentWithPermission(t *testing.T) {
	f := newFixtureWithStdin()
	defer f.Close()

	writeLine(f.PW, map[string]any{
		"type": "control_response",
		"response": map[string]any{"subtype": "success", "request_id": "init_1"},
	})
	f.Drain()

	go func() {
		scanner := bufio.NewScanner(f.StdinPR)
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg["type"] == "control_request" {
				writeLine(f.PW, map[string]any{
					"type": "control_response",
					"response": map[string]any{
						"subtype":    "success",
						"request_id": msg["request_id"],
						"response":   map[string]any{},
					},
				})
				time.Sleep(10 * time.Millisecond)
				writeLine(f.PW, map[string]any{
					"type":       "control_request",
					"request_id": "perm-incoming-1",
					"request": map[string]any{
						"subtype":   "can_use_tool",
						"tool_name": "Edit",
						"input":     map[string]any{},
					},
				})
				return
			}
		}
	}()

	_, err := f.transport.SendControlRequest("", "interrupt", nil)
	require.NoError(t, err)

	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Len(t, f.PermissionRequests, 1)
	assert.Equal(t, "Edit", f.PermissionRequests[0].ToolName)
}
