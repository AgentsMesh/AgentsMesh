package tools

import (
	"context"
	"os"
	"testing"
)

func TestNewTextResult(t *testing.T) {
	result := NewTextResult("test message")

	if len(result.Content) != 1 {
		t.Errorf("content length: got %v, want 1", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("content type: got %v, want text", result.Content[0].Type)
	}

	if result.Content[0].Text != "test message" {
		t.Errorf("content text: got %v, want 'test message'", result.Content[0].Text)
	}

	if result.IsError {
		t.Error("IsError should be false")
	}
}

func TestNewErrorResult(t *testing.T) {
	result := NewErrorResult(os.ErrNotExist)

	if len(result.Content) != 1 {
		t.Errorf("content length: got %v, want 1", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("content type: got %v, want text", result.Content[0].Type)
	}

	if !result.IsError {
		t.Error("IsError should be true")
	}
}

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Fatal("NewToolRegistry returned nil")
		return
	}

	// Registry should be empty initially (no built-in tools)
	tools := registry.List()
	if len(tools) != 0 {
		t.Errorf("registry should be empty, got %d tools", len(tools))
	}
}

func TestToolRegistryRegister(t *testing.T) {
	registry := NewToolRegistry()

	customTool := &Tool{
		Name:        "custom_tool",
		Description: "A custom tool",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return NewTextResult("custom"), nil
		},
	}

	registry.Register(customTool)

	tool, ok := registry.Get("custom_tool")
	if !ok {
		t.Error("custom_tool should be registered")
	}

	if tool.Description != "A custom tool" {
		t.Errorf("Description: got %v, want 'A custom tool'", tool.Description)
	}
}

func TestToolRegistryGetNotFound(t *testing.T) {
	registry := NewToolRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("nonexistent tool should not exist")
	}
}

func TestToolRegistryInvoke(t *testing.T) {
	registry := NewToolRegistry()

	testTool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return NewTextResult("success"), nil
		},
	}
	registry.Register(testTool)

	result, err := registry.Invoke(context.Background(), "test_tool", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content[0].Text)
	}

	if result.Content[0].Text != "success" {
		t.Errorf("content: got %v, want 'success'", result.Content[0].Text)
	}
}

func TestToolRegistryInvokeNotFound(t *testing.T) {
	registry := NewToolRegistry()

	_, err := registry.Invoke(context.Background(), "nonexistent", map[string]interface{}{})

	if err == nil {
		t.Error("should return error for nonexistent tool")
	}
}

func TestToolRegistryInvokeHandlerError(t *testing.T) {
	registry := NewToolRegistry()

	errorTool := &Tool{
		Name:        "error_tool",
		Description: "A tool that errors",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return nil, os.ErrPermission
		},
	}
	registry.Register(errorTool)

	result, err := registry.Invoke(context.Background(), "error_tool", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("should return error result")
	}
}

func TestToolRegistryInvokeWithToolResult(t *testing.T) {
	registry := NewToolRegistry()

	customTool := &Tool{
		Name:        "custom_result_tool",
		Description: "A tool that returns custom result",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return &ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "custom"}},
				IsError: false,
			}, nil
		},
	}
	registry.Register(customTool)

	result, err := registry.Invoke(context.Background(), "custom_result_tool", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Content[0].Text != "custom" {
		t.Errorf("Content: got %v, want custom", result.Content[0].Text)
	}
}

func TestToolRegistryInvokeWithJSONResult(t *testing.T) {
	registry := NewToolRegistry()

	jsonTool := &Tool{
		Name:        "json_tool",
		Description: "A tool that returns a struct",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return map[string]string{"key": "value"}, nil
		},
	}
	registry.Register(jsonTool)

	result, err := registry.Invoke(context.Background(), "json_tool", map[string]interface{}{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content[0].Text)
	}

	// Result should be JSON encoded
	if result.Content[0].Text == "" {
		t.Error("content should not be empty")
	}
}

func TestToolStruct(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{"type": "string"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return NewTextResult("success"), nil
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("Name: got %v, want test_tool", tool.Name)
	}

	if tool.Handler == nil {
		t.Error("Handler should not be nil")
	}
}

func TestContentBlock(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: "Hello world",
	}

	if block.Type != "text" {
		t.Errorf("Type: got %v, want text", block.Type)
	}

	if block.Text != "Hello world" {
		t.Errorf("Text: got %v, want 'Hello world'", block.Text)
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "Line 1"},
			{Type: "text", Text: "Line 2"},
		},
		IsError: false,
	}

	if len(result.Content) != 2 {
		t.Errorf("Content length: got %v, want 2", len(result.Content))
	}

	if result.IsError {
		t.Error("IsError should be false")
	}
}

func TestToolRegistryList(t *testing.T) {
	registry := NewToolRegistry()

	// Register multiple tools
	for i := 0; i < 3; i++ {
		tool := &Tool{
			Name:        "tool_" + string(rune('a'+i)),
			Description: "Tool " + string(rune('a'+i)),
			InputSchema: map[string]interface{}{"type": "object"},
			Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				return NewTextResult("ok"), nil
			},
		}
		registry.Register(tool)
	}

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("List length: got %v, want 3", len(tools))
	}
}
