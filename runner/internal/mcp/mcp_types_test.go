package mcp

import (
	"encoding/json"
	"testing"
)

// --- Test Config Struct ---

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		Name:    "test-server",
		Command: testDummyCmd(),
		Args:    []string{"server.js"},
		Env:     map[string]string{"NODE_ENV": "production"},
	}

	if cfg.Name != "test-server" {
		t.Errorf("Name: got %v, want test-server", cfg.Name)
	}

	if cfg.Command != testDummyCmd() {
		t.Errorf("Command: got %v, want %v", cfg.Command, testDummyCmd())
	}

	if len(cfg.Args) != 1 {
		t.Errorf("Args length: got %v, want 1", len(cfg.Args))
	}

	if cfg.Env["NODE_ENV"] != "production" {
		t.Errorf("Env[NODE_ENV]: got %v, want production", cfg.Env["NODE_ENV"])
	}
}

func TestConfigJSON(t *testing.T) {
	jsonStr := `{
		"name": "test-server",
		"command": "/usr/bin/node",
		"args": ["server.js"],
		"env": {"NODE_ENV": "production"}
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if cfg.Name != "test-server" {
		t.Errorf("Name: got %v, want test-server", cfg.Name)
	}

	if cfg.Command != "/usr/bin/node" {
		t.Errorf("Command: got %v, want /usr/bin/node", cfg.Command)
	}
}

// --- Test Tool Struct ---

func TestToolStruct(t *testing.T) {
	tool := Tool{
		Name:        "read_file",
		Description: "Read a file from disk",
		InputSchema: []byte(`{"type": "object"}`),
	}

	if tool.Name != "read_file" {
		t.Errorf("Name: got %v, want read_file", tool.Name)
	}

	if tool.Description != "Read a file from disk" {
		t.Errorf("Description: got %v, want Read a file from disk", tool.Description)
	}
}

func TestToolJSON(t *testing.T) {
	jsonStr := `{
		"name": "read_file",
		"description": "Read a file",
		"inputSchema": {"type": "object"}
	}`

	var tool Tool
	if err := json.Unmarshal([]byte(jsonStr), &tool); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if tool.Name != "read_file" {
		t.Errorf("Name: got %v, want read_file", tool.Name)
	}
}

// --- Test Resource Struct ---

func TestResourceStruct(t *testing.T) {
	res := Resource{
		URI:         "file:///tmp/test.txt",
		Name:        "test.txt",
		Description: "Test file",
		MimeType:    "text/plain",
	}

	if res.URI != "file:///tmp/test.txt" {
		t.Errorf("URI: got %v, want file:///tmp/test.txt", res.URI)
	}

	if res.Name != "test.txt" {
		t.Errorf("Name: got %v, want test.txt", res.Name)
	}

	if res.MimeType != "text/plain" {
		t.Errorf("MimeType: got %v, want text/plain", res.MimeType)
	}
}

func TestResourceJSON(t *testing.T) {
	jsonStr := `{
		"uri": "file:///tmp/test.txt",
		"name": "test.txt",
		"mimeType": "text/plain"
	}`

	var res Resource
	if err := json.Unmarshal([]byte(jsonStr), &res); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if res.URI != "file:///tmp/test.txt" {
		t.Errorf("URI: got %v, want file:///tmp/test.txt", res.URI)
	}
}

// --- Test Request Struct ---

func TestRequestStruct(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  []byte(`{}`),
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %v, want 2.0", req.JSONRPC)
	}

	if req.ID != 1 {
		t.Errorf("ID: got %v, want 1", req.ID)
	}

	if req.Method != "tools/list" {
		t.Errorf("Method: got %v, want tools/list", req.Method)
	}
}

func TestRequestJSON(t *testing.T) {
	jsonStr := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list",
		"params": {}
	}`

	var req Request
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if req.Method != "tools/list" {
		t.Errorf("Method: got %v, want tools/list", req.Method)
	}
}

// --- Test Response Struct ---

func TestResponseStruct(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  []byte(`{"tools": []}`),
		Error:   nil,
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC: got %v, want 2.0", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("ID: got %v, want 1", resp.ID)
	}

	if resp.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestResponseWithError(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  nil,
		Error: &Error{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	if resp.Error == nil {
		t.Error("Error should not be nil")
	}

	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code: got %v, want -32600", resp.Error.Code)
	}

	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message: got %v, want Invalid Request", resp.Error.Message)
	}
}

// --- Test Error Struct ---

func TestErrorStruct(t *testing.T) {
	err := Error{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    []byte(`"additional info"`),
	}

	if err.Code != -32600 {
		t.Errorf("Code: got %v, want -32600", err.Code)
	}

	if err.Message != "Invalid Request" {
		t.Errorf("Message: got %v, want Invalid Request", err.Message)
	}
}

func TestErrorJSON(t *testing.T) {
	jsonStr := `{
		"code": -32600,
		"message": "Invalid Request",
		"data": "extra info"
	}`

	var e Error
	if err := json.Unmarshal([]byte(jsonStr), &e); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if e.Code != -32600 {
		t.Errorf("Code: got %v, want -32600", e.Code)
	}
}
