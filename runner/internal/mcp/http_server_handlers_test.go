package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPServerHealth(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code: got %v, want %v", rec.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("status: got %v, want ok", result["status"])
	}
}

func TestHTTPServerPods(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("pod-1", "test-org", nil, nil, "claude")

	req := httptest.NewRequest(http.MethodGet, "/pods", nil)
	rec := httptest.NewRecorder()

	server.handlePods(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code: got %v, want %v", rec.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	pods, ok := result["pods"].([]interface{})
	if !ok || len(pods) != 1 {
		t.Errorf("pods: got %v, want 1 pod", pods)
	}
}

func TestHTTPServerMCPMissingPodKey(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("should return error for missing pod key")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("error code: got %v, want -32600", resp.Error.Code)
	}
}

func TestHTTPServerMCPUnregisteredPod(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "unknown-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("should return error for unregistered pod")
	}
}

func TestHTTPServerMCPMethodNotFound(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"unknown/method"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("should return error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("error code: got %v, want -32601", resp.Error.Code)
	}
}

func TestHTTPServerMCPInvalidJSON(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("should return error for invalid JSON")
	}

	if resp.Error.Code != -32700 {
		t.Errorf("error code: got %v, want -32700", resp.Error.Code)
	}
}

func TestHTTPServerMCPMethodNotAllowed(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code: got %v, want %v", rec.Code, http.StatusMethodNotAllowed)
	}
}
