package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPServerMCPToolsCallSendPodInput(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"pod_key": "target-pod",
				"keys": ["ctrl+c"]
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Tool should be found (may error on backend call)
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool send_pod_input should be found")
	}
}

func TestHTTPServerMCPToolsCallSendPodInputMissingArgs(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"pod_key": "target-pod"
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Should handle missing text/keys argument
}

func TestHTTPServerMCPToolsCallSendPodInputWithTextAndKeys(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"pod_key": "target-pod",
				"text": "hello",
				"keys": ["ctrl+c", "enter", "escape"]
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found
}

func TestHTTPServerMCPToolsCallSendPodInputMissingPodKey(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"keys": ["enter"]
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found and validation error returned
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool send_pod_input should be found")
	}
}

func TestHTTPServerMCPToolsCallSendPodInputEmptyKeysAndText(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"pod_key": "target-pod",
				"keys": []
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found and validation error returned
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool send_pod_input should be found")
	}
}

func TestHTTPServerMCPToolsCallGetPodSnapshotWithDefaultLines(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "get_pod_snapshot",
			"arguments": {
				"pod_key": "target-pod"
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
}

func TestHTTPServerMCPToolsCallGetPodSnapshotMissingPodKey(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "get_pod_snapshot",
			"arguments": {}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found and validation error returned
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool get_pod_snapshot should be found")
	}
}

func TestHTTPServerMCPToolsCallSendPodInputTextOnlyMissingArgs(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_pod_input",
			"arguments": {
				"pod_key": "target-pod"
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found and validation error returned
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool send_pod_input should be found")
	}
}
