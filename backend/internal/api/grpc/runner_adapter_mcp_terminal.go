package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// TerminalRouterForMCP defines the interface for terminal router operations needed by MCP handlers.
type TerminalRouterForMCP interface {
	GetRunnerID(podKey string) (int64, bool)
	RouteInput(podKey string, data []byte) error
}

// ==================== Terminal MCP Methods ====================

// mcpObserveTerminal handles the "observe_terminal" MCP method.
// Proxies the request to the Runner via gRPC and waits for the result.
func (a *GRPCRunnerAdapter) mcpObserveTerminal(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
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

	// Look up runner ID from terminal router
	tr, ok := a.terminalRouter.(TerminalRouterForMCP)
	if !ok || tr == nil {
		return nil, newMcpError(503, "terminal router not available")
	}

	runnerID, found := tr.GetRunnerID(params.PodKey)
	if !found {
		return nil, newMcpError(404, "pod not registered on any runner")
	}

	// Ensure terminal query service is available
	if a.terminalQueryService == nil {
		return nil, newMcpError(503, "terminal query service not available")
	}

	lines := params.Lines
	if lines <= 0 {
		lines = 100
	}
	if lines == -1 {
		lines = 10000
	}

	// Proxy the request to the runner and wait for the result
	result, err := a.terminalQueryService.ObserveTerminal(
		ctx,
		runnerID,
		params.PodKey,
		lines,
		params.IncludeScreen,
		a.SendObserveTerminal,
	)
	if err != nil {
		return nil, newMcpErrorf(500, "failed to observe terminal: %v", err)
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

// mcpSendTerminalText handles the "send_terminal_text" MCP method.
func (a *GRPCRunnerAdapter) mcpSendTerminalText(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		PodKey string `json:"pod_key"`
		Text   string `json:"text"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.PodKey == "" {
		return nil, newMcpError(400, "pod_key is required")
	}
	if params.Text == "" {
		return nil, newMcpError(400, "text is required")
	}

	// Verify pod belongs to the organization
	pod, err := a.podService.GetPodByKey(ctx, params.PodKey)
	if err != nil {
		return nil, newMcpError(404, "pod not found")
	}
	if pod.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	tr, ok := a.terminalRouter.(TerminalRouterForMCP)
	if !ok || tr == nil {
		return nil, newMcpError(503, "terminal router not available")
	}

	if err := tr.RouteInput(params.PodKey, []byte(params.Text)); err != nil {
		return nil, newMcpErrorf(500, "failed to send terminal text: %v", err)
	}

	return map[string]interface{}{"message": "text sent"}, nil
}

// mcpSendTerminalKey handles the "send_terminal_key" MCP method.
func (a *GRPCRunnerAdapter) mcpSendTerminalKey(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		PodKey string   `json:"pod_key"`
		Keys   []string `json:"keys"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.PodKey == "" {
		return nil, newMcpError(400, "pod_key is required")
	}
	if len(params.Keys) == 0 {
		return nil, newMcpError(400, "keys is required")
	}

	// Verify pod belongs to the organization
	pod, err := a.podService.GetPodByKey(ctx, params.PodKey)
	if err != nil {
		return nil, newMcpError(404, "pod not found")
	}
	if pod.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}

	tr, ok := a.terminalRouter.(TerminalRouterForMCP)
	if !ok || tr == nil {
		return nil, newMcpError(503, "terminal router not available")
	}

	// Convert key names to terminal escape sequences
	for _, key := range params.Keys {
		input := convertKeyToInput(key)
		if err := tr.RouteInput(params.PodKey, []byte(input)); err != nil {
			return nil, newMcpErrorf(500, "failed to send terminal key: %v", err)
		}
	}

	return map[string]interface{}{"message": "keys sent"}, nil
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
