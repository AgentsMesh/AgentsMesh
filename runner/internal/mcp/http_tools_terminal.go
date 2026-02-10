package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
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

			// Try local terminal provider first (for AutopilotController control process)
			if s.terminalProvider != nil {
				output, err := s.terminalProvider.GetTerminalOutput(podKey, lines)
				if err == nil {
					return output, nil
				}
				// Fall through to Backend API if local access fails
			}

			// Fall back to Backend API for remote pods
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

			// Try local terminal provider first (for AutopilotController control process)
			if s.terminalProvider != nil {
				err := s.terminalProvider.SendTerminalText(podKey, text)
				if err == nil {
					return "Text sent successfully", nil
				}
				// Fall through to Backend API if local access fails
			}

			// Fall back to Backend API for remote pods
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

			// Try local terminal provider first (for AutopilotController control process)
			if s.terminalProvider != nil {
				err := s.terminalProvider.SendTerminalKey(podKey, keys)
				if err == nil {
					return "Keys sent successfully", nil
				}
				// Fall through to Backend API if local access fails
			}

			// Fall back to Backend API for remote pods
			err := client.SendTerminalKey(ctx, podKey, keys)
			if err != nil {
				return nil, err
			}
			return "Keys sent successfully", nil
		},
	}
}

// PodStatusResult represents the result of get_pod_status tool.
type PodStatusResult struct {
	PodKey      string `json:"pod_key"`
	AgentStatus string `json:"agent_status"` // executing, waiting, idle
	PodStatus   string `json:"pod_status"`   // running, completed, error, etc.
	Message     string `json:"message,omitempty"`
}

func (s *HTTPServer) createGetPodStatusTool() *MCPTool {
	return &MCPTool{
		Name:        "get_pod_status",
		Description: "Get the agent execution status of a pod. Returns: executing (agent is actively running commands), waiting (agent is waiting for user input), idle (agent is not actively running).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod_key": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod to check status",
				},
			},
			"required": []string{"pod_key"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			podKey := getStringArg(args, "pod_key")
			if podKey == "" {
				return nil, fmt.Errorf("pod_key is required")
			}

			// Check if status provider is available
			if s.statusProvider == nil {
				return PodStatusResult{
					PodKey:      podKey,
					AgentStatus: "idle",
					PodStatus:   "unknown",
					Message:     "Status provider not configured",
				}, nil
			}

			// Get status from provider
			agentStatus, podStatus, _, found := s.statusProvider.GetPodStatus(podKey)
			if !found {
				return PodStatusResult{
					PodKey:      podKey,
					AgentStatus: "idle",
					PodStatus:   "not_found",
					Message:     "Pod not found",
				}, nil
			}

			result := PodStatusResult{
				PodKey:      podKey,
				AgentStatus: agentStatus,
				PodStatus:   podStatus,
			}

			// Return as JSON for structured output
			jsonBytes, err := json.Marshal(result)
			if err != nil {
				return result, nil
			}
			return string(jsonBytes), nil
		},
	}
}
