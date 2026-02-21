package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

func TestHTTPServerMCPToolsCallCreatePod(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"ticket_id": 123,
				"command": "echo hello"
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

func TestHTTPServerMCPToolsCallCreatePodWithAllParams(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"agent_type_id": 1,
				"runner_id": 2,
				"ticket_id": 123,
				"initial_prompt": "Hello, start working on this task",
				"model": "claude-opus-4",
				"repository_id": 456,
				"branch_name": "feature/new-feature",
				"credential_profile_id": 789,
				"permission_mode": "plan",
				"config_overrides": {
					"timeout": 300,
					"max_tokens": 4096
				}
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found (may error on backend call, but params should be parsed)
}

func TestHTTPServerMCPToolsCallCreatePodWithRepositoryURL(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"agent_type_id": 1,
				"repository_url": "https://github.com/example/repo.git",
				"branch_name": "main"
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

func TestHTTPServerMCPToolsCallCreatePodWithBypassPermissions(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"agent_type_id": 1,
				"permission_mode": "bypassPermissions"
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

func TestHTTPServerMCPToolsCallCreatePodWithEmptyConfigOverrides(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"agent_type_id": 1,
				"config_overrides": {}
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

func TestMergeModelIntoConfigOverrides(t *testing.T) {
	t.Run("merges model into nil config_overrides", func(t *testing.T) {
		req := &tools.PodCreateRequest{}
		mergeModelIntoConfigOverrides(req, "sonnet")

		if req.ConfigOverrides == nil {
			t.Fatal("ConfigOverrides should not be nil")
		}
		if req.ConfigOverrides["model"] != "sonnet" {
			t.Errorf("model = %v, want sonnet", req.ConfigOverrides["model"])
		}
	})

	t.Run("merges model into existing config_overrides without model", func(t *testing.T) {
		req := &tools.PodCreateRequest{
			ConfigOverrides: map[string]interface{}{
				"timeout": 300,
			},
		}
		mergeModelIntoConfigOverrides(req, "opus")

		if req.ConfigOverrides["model"] != "opus" {
			t.Errorf("model = %v, want opus", req.ConfigOverrides["model"])
		}
		if req.ConfigOverrides["timeout"] != 300 {
			t.Errorf("timeout = %v, want 300 (should be preserved)", req.ConfigOverrides["timeout"])
		}
	})

	t.Run("does not override model already in config_overrides", func(t *testing.T) {
		req := &tools.PodCreateRequest{
			ConfigOverrides: map[string]interface{}{
				"model": "haiku",
			},
		}
		mergeModelIntoConfigOverrides(req, "opus")

		if req.ConfigOverrides["model"] != "haiku" {
			t.Errorf("model = %v, want haiku (should not be overridden)", req.ConfigOverrides["model"])
		}
	})

	t.Run("skips merge when model is empty string", func(t *testing.T) {
		req := &tools.PodCreateRequest{}
		mergeModelIntoConfigOverrides(req, "")

		if req.ConfigOverrides != nil {
			t.Errorf("ConfigOverrides should remain nil when model is empty, got %v", req.ConfigOverrides)
		}
	})
}

func TestHTTPServerMCPToolsCallCreatePodMissingAgentTypeID(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"initial_prompt": "Hello"
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
		t.Error("tool create_pod should be found")
	}
}
