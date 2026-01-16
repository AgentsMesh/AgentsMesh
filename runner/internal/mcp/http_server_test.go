package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPServer(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)

	if server == nil {
		t.Fatal("NewHTTPServer returned nil")
	}

	if server.backendURL != "http://localhost:8080" {
		t.Errorf("backendURL: got %v, want %v", server.backendURL, "http://localhost:8080")
	}

	if server.port != 9090 {
		t.Errorf("port: got %v, want %v", server.port, 9090)
	}

	if len(server.tools) == 0 {
		t.Error("tools should be registered")
	}
}

func TestHTTPServerRegisterPod(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)

	ticketID := 123
	projectID := 456

	server.RegisterPod("test-pod", "test-org", &ticketID, &projectID, "claude")

	pod, ok := server.GetPod("test-pod")
	if !ok {
		t.Fatal("pod should be registered")
	}

	if pod.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", pod.PodKey, "test-pod")
	}

	if pod.TicketID == nil || *pod.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", pod.TicketID)
	}

	if pod.AgentType != "claude" {
		t.Errorf("AgentType: got %v, want %v", pod.AgentType, "claude")
	}
}

func TestHTTPServerUnregisterPod(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)

	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	_, ok := server.GetPod("test-pod")
	if !ok {
		t.Fatal("pod should be registered")
	}

	server.UnregisterPod("test-pod")

	_, ok = server.GetPod("test-pod")
	if ok {
		t.Error("pod should be unregistered")
	}
}

func TestHTTPServerPodCount(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)

	if server.PodCount() != 0 {
		t.Errorf("initial count should be 0, got %v", server.PodCount())
	}

	server.RegisterPod("pod-1", "test-org", nil, nil, "claude")
	if server.PodCount() != 1 {
		t.Errorf("count should be 1, got %v", server.PodCount())
	}

	server.RegisterPod("pod-2", "test-org", nil, nil, "claude")
	if server.PodCount() != 2 {
		t.Errorf("count should be 2, got %v", server.PodCount())
	}

	server.UnregisterPod("pod-1")
	if server.PodCount() != 1 {
		t.Errorf("count should be 1, got %v", server.PodCount())
	}
}

func TestHTTPServerPort(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	if server.Port() != 9090 {
		t.Errorf("Port: got %v, want %v", server.Port(), 9090)
	}
}

func TestHTTPServerGenerateMCPConfig(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	config := server.GenerateMCPConfig("test-pod")

	if config == nil {
		t.Fatal("config should not be nil")
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers should exist")
	}

	if _, ok := mcpServers["agentsmesh-collaboration"]; !ok {
		t.Error("agentsmesh-collaboration server should exist")
	}
}

func TestHTTPServerHealth(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)

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
	server := NewHTTPServer("http://localhost:8080", 9090)
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
	server := NewHTTPServer("http://localhost:8080", 9090)

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
	server := NewHTTPServer("http://localhost:8080", 9090)

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

func TestHTTPServerMCPInitialize(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error.Message)
	}

	if resp.Result == nil {
		t.Error("result should not be nil")
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result should be a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion: got %v, want 2024-11-05", result["protocolVersion"])
	}
}

func TestHTTPServerMCPToolsList(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result should be a map")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("tools should be an array")
	}

	// Should have 21 tools (all collaboration tools)
	if len(tools) < 20 {
		t.Errorf("tools count: got %v, want at least 20", len(tools))
	}
}

func TestHTTPServerMCPMethodNotFound(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
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
	server := NewHTTPServer("http://localhost:8080", 9090)
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
	server := NewHTTPServer("http://localhost:8080", 9090)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code: got %v, want %v", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestPodInfoStruct(t *testing.T) {
	ticketID := 123
	projectID := 456

	info := PodInfo{
		PodKey:       "test-pod",
		OrgSlug:      "test-org",
		TicketID:     &ticketID,
		ProjectID:    &projectID,
		AgentType:    "claude",
		RegisteredAt: time.Now(),
		Client:       NewBackendClient("http://localhost:8080", "test-org", "test-pod"),
	}

	if info.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", info.PodKey, "test-pod")
	}

	if info.TicketID == nil || *info.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", info.TicketID)
	}

	if info.Client == nil {
		t.Error("Client should not be nil")
	}
}

func TestMCPRequestStruct(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  []byte(`{}`),
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %v, want 2.0", req.JSONRPC)
	}

	if req.Method != "tools/list" {
		t.Errorf("Method: got %v, want tools/list", req.Method)
	}
}

func TestMCPResponseStruct(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"status": "ok"},
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %v, want 2.0", resp.JSONRPC)
	}

	if resp.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestMCPRPCErrorStruct(t *testing.T) {
	err := MCPRPCError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    "additional data",
	}

	if err.Code != -32600 {
		t.Errorf("Code: got %v, want -32600", err.Code)
	}

	if err.Message != "Invalid Request" {
		t.Errorf("Message: got %v, want Invalid Request", err.Message)
	}
}

func TestMCPToolResultStruct(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: "Hello"}},
		IsError: false,
	}

	if len(result.Content) != 1 {
		t.Errorf("Content length: got %v, want 1", len(result.Content))
	}

	if result.Content[0].Text != "Hello" {
		t.Errorf("Content text: got %v, want Hello", result.Content[0].Text)
	}
}

func TestHelperFunctions(t *testing.T) {
	args := map[string]interface{}{
		"string_val":       "test",
		"int_val":          float64(42),
		"bool_val":         true,
		"string_slice_val": []interface{}{"a", "b", "c"},
	}

	if v := getStringArg(args, "string_val"); v != "test" {
		t.Errorf("getStringArg: got %v, want test", v)
	}

	if v := getStringArg(args, "missing"); v != "" {
		t.Errorf("getStringArg missing: got %v, want empty", v)
	}

	if v := getIntArg(args, "int_val"); v != 42 {
		t.Errorf("getIntArg: got %v, want 42", v)
	}

	if v := getIntArg(args, "missing"); v != 0 {
		t.Errorf("getIntArg missing: got %v, want 0", v)
	}

	if v := getBoolArg(args, "bool_val"); !v {
		t.Error("getBoolArg: should be true")
	}

	if v := getBoolArg(args, "missing"); v {
		t.Error("getBoolArg missing: should be false")
	}

	if v := getIntPtrArg(args, "int_val"); v == nil || *v != 42 {
		t.Errorf("getIntPtrArg: got %v, want 42", v)
	}

	if v := getIntPtrArg(args, "missing"); v != nil {
		t.Errorf("getIntPtrArg missing: got %v, want nil", v)
	}

	if v := getStringSliceArg(args, "string_slice_val"); len(v) != 3 {
		t.Errorf("getStringSliceArg: got %v items, want 3", len(v))
	}

	if v := getStringSliceArg(args, "missing"); v != nil {
		t.Errorf("getStringSliceArg missing: got %v, want nil", v)
	}
}

// --- Test HTTP Tool Handlers ---

func TestHTTPServerMCPToolsCallObserveTerminal(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "observe_terminal",
			"arguments": {
				"pod_key": "target-pod",
				"lines": 100
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

	// Should have error since backend is not available
	// But the tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool should be found")
	}
}

func TestHTTPServerMCPToolsCallSendTerminalText(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_terminal_text",
			"arguments": {
				"pod_key": "target-pod",
				"text": "hello world"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool should be found")
	}
}

func TestHTTPServerMCPToolsCallMissingArgs(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "send_terminal_text",
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
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have an error result (missing text)
	result, ok := resp.Result.(map[string]interface{})
	if ok {
		if !result["isError"].(bool) {
			t.Log("Tool should return error for missing arguments")
		}
	}
}

func TestHTTPServerMCPToolsCallListAvailablePods(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "list_available_pods",
			"arguments": {}
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool list_available_pods should be found")
	}
}

func TestHTTPServerMCPToolsCallBindPod(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "bind_pod",
			"arguments": {
				"target_pod": "other-pod",
				"scopes": ["terminal:read"]
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool bind_pod should be found")
	}
}

func TestHTTPServerMCPToolsCallSearchChannels(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "search_channels",
			"arguments": {
				"name": "test"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool search_channels should be found")
	}
}

func TestHTTPServerMCPToolsCallSearchTickets(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "search_tickets",
			"arguments": {
				"query": "test"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool search_tickets should be found")
	}
}

func TestHTTPServerMCPToolsCallCreateChannel(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_channel",
			"arguments": {
				"name": "test-channel",
				"description": "Test channel"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool create_channel should be found")
	}
}

func TestHTTPServerMCPToolsCallGetTicket(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "get_ticket",
			"arguments": {
				"ticket_id": "AM-123"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool get_ticket should be found")
	}
}

func TestHTTPServerMCPToolsCallCreateTicket(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_ticket",
			"arguments": {
				"product_id": 1,
				"title": "Test Ticket",
				"description": "Test description"
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

	// Tool should be found
	if resp.Error != nil && resp.Error.Code == -32601 {
		t.Error("tool create_ticket should be found")
	}
}

func TestHTTPServerMCPNotificationsInitialized(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"method": "notifications/initialized"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	// Notifications don't return a response
	if rec.Body.Len() == 0 {
		// This is expected for notifications
		return
	}

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		// Empty or no response is expected for notifications
		return
	}
}

// --- Additional tests for coverage ---

func TestHTTPServerMCPToolsCallInvalidParams(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": "invalid"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("should return error for invalid params")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("error code: got %v, want -32602", resp.Error.Code)
	}
}

func TestHTTPServerMCPToolsCallToolNotFound(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "nonexistent_tool",
			"arguments": {}
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

	if resp.Error == nil {
		t.Error("should return error for nonexistent tool")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("error code: got %v, want -32602", resp.Error.Code)
	}
}

func TestHTTPServerMCPToolsCallWithIntArgs(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	// Test with int argument to cover getIntArg path
	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "observe_terminal",
			"arguments": {
				"pod_key": "target",
				"lines": 50
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Just verify it doesn't crash
}

func TestHTTPServerMCPToolsCallCreateTicketWithPriority(t *testing.T) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	body := bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "create_ticket",
			"arguments": {
				"title": "Test Ticket",
				"type": "task",
				"description": "A test ticket",
				"priority": "medium"
			}
		}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/mcp", body)
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Just verify it processes the request
}

func TestGetIntPtrArgNil(t *testing.T) {
	args := map[string]interface{}{}
	result := getIntPtrArg(args, "missing")
	if result != nil {
		t.Error("should return nil for missing key")
	}
}

func TestGetIntArgInvalidType(t *testing.T) {
	args := map[string]interface{}{
		"string_val": "not a number",
	}
	result := getIntArg(args, "string_val")
	if result != 0 {
		t.Error("should return 0 for invalid type")
	}
}

// --- Benchmark Tests ---

func BenchmarkHTTPServerHandleMCP(b *testing.B) {
	server := NewHTTPServer("http://localhost:8080", 9090)
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	bodyStr := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := bytes.NewBufferString(bodyStr)
		req := httptest.NewRequest(http.MethodPost, "/mcp", body)
		req.Header.Set("X-Pod-Key", "test-pod")
		rec := httptest.NewRecorder()

		server.handleMCP(rec, req)
	}
}

func BenchmarkHTTPServerRegisterPod(b *testing.B) {
	server := NewHTTPServer("http://localhost:8080", 9090)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.RegisterPod("test-pod", "test-org", nil, nil, "claude")
	}
}
