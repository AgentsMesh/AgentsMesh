package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
				"ticket_slug": "AM-123",
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
				"agent_slug": "claude-code",
				"runner_id": 2,
				"ticket_slug": "AM-123",
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
				"agent_slug": "claude-code",
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
				"agent_slug": "claude-code",
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
				"agent_slug": "claude-code",
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

func TestBuildAgentfileLayerFromArgs(t *testing.T) {
	t.Run("generates CONFIG declarations from args", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("opus", "bypassPermissions", "", map[string]interface{}{"timeout": float64(300)}, "", "")

		if !strings.Contains(layer, `CONFIG model = "opus"`) {
			t.Errorf("missing model CONFIG, got: %s", layer)
		}
		if !strings.Contains(layer, `CONFIG permission_mode = "bypassPermissions"`) {
			t.Errorf("missing permission_mode CONFIG, got: %s", layer)
		}
		if !strings.Contains(layer, `CONFIG timeout = 300`) {
			t.Errorf("missing timeout CONFIG, got: %s", layer)
		}
	})

	t.Run("generates PROMPT declaration", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("", "", "fix the bug", nil, "", "")

		if !strings.Contains(layer, `PROMPT "fix the bug"`) {
			t.Errorf("missing PROMPT, got: %s", layer)
		}
	})

	t.Run("skips empty args", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("", "", "", nil, "", "")
		if layer != "" {
			t.Errorf("expected empty layer, got: %s", layer)
		}
	})

	t.Run("deduplicates model and permission_mode from config_overrides", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("opus", "plan", "", map[string]interface{}{
			"model":           "haiku",
			"permission_mode": "default",
			"mcp_enabled":     true,
		}, "", "")

		modelCount := strings.Count(layer, "CONFIG model")
		if modelCount != 1 {
			t.Errorf("expected 1 CONFIG model, got %d in: %s", modelCount, layer)
		}
		if !strings.Contains(layer, `CONFIG mcp_enabled = true`) {
			t.Errorf("missing mcp_enabled, got: %s", layer)
		}
	})

	t.Run("escapes newlines and tabs in prompt", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("", "", "line1\nline2\ttab", nil, "", "")
		if !strings.Contains(layer, `PROMPT "line1\nline2\ttab"`) {
			t.Errorf("prompt escape failed, got: %s", layer)
		}
	})

	t.Run("escapes newlines and tabs in config string values", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("", "", "", map[string]interface{}{
			"custom": "val\nwith\tnewline",
		}, "", "")
		if !strings.Contains(layer, `CONFIG custom = "val\nwith\tnewline"`) {
			t.Errorf("config escape failed, got: %s", layer)
		}
	})

	t.Run("generates REPO and BRANCH declarations", func(t *testing.T) {
		layer := buildAgentfileLayerFromArgs("", "", "", nil, "https://github.com/example/repo.git", "main")
		if !strings.Contains(layer, `REPO "https://github.com/example/repo.git"`) {
			t.Errorf("missing REPO, got: %s", layer)
		}
		if !strings.Contains(layer, `BRANCH "main"`) {
			t.Errorf("missing BRANCH, got: %s", layer)
		}
	})
}

func TestHTTPServerMCPToolsCallCreatePodWithAlias(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_pod",
			"arguments": {
				"agent_slug": "claude-code",
				"runner_id": 2,
				"alias": "my-feature-pod",
				"initial_prompt": "Work on feature X"
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Tool should be found (may error on backend call, but alias param should be parsed)
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool create_pod should be found")
	}
}

func TestHTTPServerMCPToolsCallCreatePodMissingAgentSlug(t *testing.T) {
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
