package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentmesh/runner/internal/mcp/tools"
)

// Terminal Tools

func (s *HTTPServer) createObserveTerminalTool() *MCPTool {
	return &MCPTool{
		Name:        "observe_terminal",
		Description: "Observe the terminal output of another agent pod. Requires terminal:read permission via binding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod_key": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod to observe",
				},
				"lines": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to retrieve (default: 50)",
				},
				"raw": map[string]interface{}{
					"type":        "boolean",
					"description": "Return raw output without ANSI processing (default: false)",
				},
				"include_screen": map[string]interface{}{
					"type":        "boolean",
					"description": "Include current screen content (default: false)",
				},
			},
			"required": []string{"pod_key"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			podKey := getStringArg(args, "pod_key")
			if podKey == "" {
				return nil, fmt.Errorf("pod_key is required")
			}

			lines := getIntArg(args, "lines")
			if lines == 0 {
				lines = 50
			}
			raw := getBoolArg(args, "raw")
			includeScreen := getBoolArg(args, "include_screen")

			return client.ObserveTerminal(ctx, podKey, lines, raw, includeScreen)
		},
	}
}

func (s *HTTPServer) createSendTerminalTextTool() *MCPTool {
	return &MCPTool{
		Name:        "send_terminal_text",
		Description: "Send text input to another agent's terminal. Requires terminal:write permission via binding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod_key": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to send to the terminal",
				},
			},
			"required": []string{"pod_key", "text"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			podKey := getStringArg(args, "pod_key")
			text := getStringArg(args, "text")

			if podKey == "" || text == "" {
				return nil, fmt.Errorf("pod_key and text are required")
			}

			err := client.SendTerminalText(ctx, podKey, text)
			if err != nil {
				return nil, err
			}
			return "Text sent successfully", nil
		},
	}
}

func (s *HTTPServer) createSendTerminalKeyTool() *MCPTool {
	return &MCPTool{
		Name:        "send_terminal_key",
		Description: "Send special keys to another agent's terminal. Supports: enter, escape, tab, backspace, delete, ctrl+c, ctrl+d, ctrl+u, ctrl+l, ctrl+z, ctrl+a, ctrl+e, ctrl+k, ctrl+w, up, down, left, right, home, end, pageup, pagedown, shift+tab",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod_key": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod",
				},
				"keys": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of keys to send (e.g., ['ctrl+c', 'enter'])",
				},
			},
			"required": []string{"pod_key", "keys"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			podKey := getStringArg(args, "pod_key")
			keys := getStringSliceArg(args, "keys")

			if podKey == "" || len(keys) == 0 {
				return nil, fmt.Errorf("pod_key and keys are required")
			}

			err := client.SendTerminalKey(ctx, podKey, keys)
			if err != nil {
				return nil, err
			}
			return "Keys sent successfully", nil
		},
	}
}
