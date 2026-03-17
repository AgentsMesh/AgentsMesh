package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// TerminalRouterForMCP defines the interface for terminal router operations needed by MCP handlers.
type TerminalRouterForMCP interface {
	RouteInput(podKey string, data []byte) error
	ObserveTerminal(ctx context.Context, podKey string, lines int32, includeScreen bool) (*runner.ObserveTerminalQueryResult, error)
}

// ==================== Pod interaction MCP Methods ====================

// mcpGetPodSnapshot handles the "get_pod_snapshot" MCP method.
// Proxies the request to the Runner via gRPC and waits for the result.
func (a *GRPCRunnerAdapter) mcpGetPodSnapshot(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		PodKey        string `json:"pod_key"`
		Lines         int32  `json:"lines"`
		IncludeScreen bool   `json:"include_screen"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.PodKey == "" {
		return nil, newMcpError(400, "pod_key is required")
	}

	// Verify pod belongs to the organization
	pod, err := a.podService.GetPodByKey(ctx, params.PodKey)
	if err != nil {
		return nil, newMcpError(404, "pod not found")
	}
	if pod.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	// Look up terminal router
	if a.terminalRouter == nil {
		return nil, newMcpError(503, "terminal router not available")
	}

	lines := params.Lines
	if lines == -1 {
		lines = 10000
	}
	if lines <= 0 {
		lines = 100
	}

	// Proxy the request to the runner and wait for the result
	result, err := a.terminalRouter.ObserveTerminal(ctx, params.PodKey, lines, params.IncludeScreen)
	if err != nil {
		return nil, newMcpErrorf(500, "failed to get pod snapshot: %v", err)
	}

	if result.Error != "" {
		return nil, newMcpError(500, result.Error)
	}

	response := map[string]interface{}{
		"pod_key":     params.PodKey,
		"output":      result.Output,
		"cursor_x":    result.CursorX,
		"cursor_y":    result.CursorY,
		"total_lines": result.TotalLines,
		"has_more":    result.HasMore,
	}

	if params.IncludeScreen && result.Screen != "" {
		response["screen"] = result.Screen
	}

	return response, nil
}

// mcpSendPodInput handles the "send_pod_input" MCP method.
// Accepts text and/or keys; sends text first, then loops through keys.
func (a *GRPCRunnerAdapter) mcpSendPodInput(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		PodKey string   `json:"pod_key"`
		Text   string   `json:"text"`
		Keys   []string `json:"keys"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.PodKey == "" {
		return nil, newMcpError(400, "pod_key is required")
	}
	if params.Text == "" && len(params.Keys) == 0 {
		return nil, newMcpError(400, "at least one of text or keys is required")
	}

	// Verify pod belongs to the organization
	pod, err := a.podService.GetPodByKey(ctx, params.PodKey)
	if err != nil {
		return nil, newMcpError(404, "pod not found")
	}
	if pod.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	if a.terminalRouter == nil {
		return nil, newMcpError(503, "terminal router not available")
	}

	// Send text first
	if params.Text != "" {
		if err := a.terminalRouter.RouteInput(params.PodKey, []byte(params.Text)); err != nil {
			return nil, newMcpErrorf(500, "failed to send pod input text: %v", err)
		}
	}

	// Then loop through keys
	for _, key := range params.Keys {
		input := convertKeyToInput(key)
		if err := a.terminalRouter.RouteInput(params.PodKey, []byte(input)); err != nil {
			return nil, newMcpErrorf(500, "failed to send pod input key: %v", err)
		}
	}

	return map[string]interface{}{"message": "input sent"}, nil
}

// convertKeyToInput converts a key name to its terminal escape sequence.
func convertKeyToInput(key string) string {
	switch key {
	case "enter", "Enter":
		return "\r"
	case "tab", "Tab":
		return "\t"
	case "escape", "Escape", "esc":
		return "\x1b"
	case "backspace", "Backspace":
		return "\x7f"
	case "delete", "Delete":
		return "\x1b[3~"
	case "up", "Up", "ArrowUp":
		return "\x1b[A"
	case "down", "Down", "ArrowDown":
		return "\x1b[B"
	case "right", "Right", "ArrowRight":
		return "\x1b[C"
	case "left", "Left", "ArrowLeft":
		return "\x1b[D"
	case "home", "Home":
		return "\x1b[H"
	case "end", "End":
		return "\x1b[F"
	case "ctrl+c", "Ctrl+C":
		return "\x03"
	case "ctrl+d", "Ctrl+D":
		return "\x04"
	case "ctrl+z", "Ctrl+Z":
		return "\x1a"
	case "ctrl+l", "Ctrl+L":
		return "\x0c"
	case "ctrl+a", "Ctrl+A":
		return "\x01"
	case "ctrl+e", "Ctrl+E":
		return "\x05"
	default:
		return key
	}
}
