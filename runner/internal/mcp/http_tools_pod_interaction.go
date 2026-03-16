package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Pod Interaction Tools

func (s *HTTPServer) createGetPodSnapshotTool() *MCPTool {
	return &MCPTool{
		Name:        "get_pod_snapshot",
		Description: "Get a snapshot of another agent pod's output. Requires pod:read permission via binding.",
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

			// Try local pod provider first (for AutopilotController control process)
			if s.podProvider != nil {
				output, err := s.podProvider.GetPodSnapshot(podKey, lines)
				if err == nil {
					return output, nil
				}
				// Fall through to Backend API if local access fails
			}

			// Fall back to Backend API for remote pods
			raw := getBoolArg(args, "raw")
			includeScreen := getBoolArg(args, "include_screen")
			return client.GetPodSnapshot(ctx, podKey, lines, raw, includeScreen)
		},
	}
}

func (s *HTTPServer) createSendPodInputTool() *MCPTool {
	return &MCPTool{
		Name:        "send_pod_input",
		Description: "Send text and/or special keys to another agent pod. Requires pod:write permission via binding. At least one of text or keys must be provided. Supports keys: enter, escape, tab, backspace, delete, ctrl+c, ctrl+d, ctrl+u, ctrl+l, ctrl+z, ctrl+a, ctrl+e, ctrl+k, ctrl+w, up, down, left, right, home, end, pageup, pagedown, shift+tab",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod_key": map[string]interface{}{
					"type":        "string",
					"description": "The pod key of the target pod",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Text to send to the pod (optional if keys is provided)",
				},
				"keys": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of special keys to send (e.g., ['ctrl+c', 'enter']). Optional if text is provided.",
				},
			},
			"required": []string{"pod_key"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			podKey := getStringArg(args, "pod_key")
			if podKey == "" {
				return nil, fmt.Errorf("pod_key is required")
			}

			text := getStringArg(args, "text")
			keys := getStringSliceArg(args, "keys")

			if text == "" && len(keys) == 0 {
				return nil, fmt.Errorf("at least one of text or keys is required")
			}

			// Try local pod provider first (for AutopilotController control process)
			if s.podProvider != nil {
				err := s.podProvider.SendPodInput(podKey, text, keys)
				if err == nil {
					return "Input sent successfully", nil
				}
				// Fall through to Backend API if local access fails
			}

			// Fall back to Backend API for remote pods
			err := client.SendPodInput(ctx, podKey, text, keys)
			if err != nil {
				return nil, err
			}
			return "Input sent successfully", nil
		},
	}
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
				return fmt.Sprintf("Pod: %s | Agent: idle | Status: unknown | Status provider not configured", podKey), nil
			}

			// Get status from provider
			agentStatus, podStatus, _, found := s.statusProvider.GetPodStatus(podKey)
			if !found {
				return fmt.Sprintf("Pod: %s | Agent: idle | Status: not_found | Pod not found", podKey), nil
			}

			return fmt.Sprintf("Pod: %s | Agent: %s | Status: %s", podKey, agentStatus, podStatus), nil
		},
	}
}
