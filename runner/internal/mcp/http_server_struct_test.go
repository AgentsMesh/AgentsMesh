package mcp

import (
	"testing"
	"time"
)

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
		// Client is a tools.CollaborationClient interface; nil is valid for struct test
	}

	if info.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", info.PodKey, "test-pod")
	}

	if info.TicketID == nil || *info.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", info.TicketID)
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
