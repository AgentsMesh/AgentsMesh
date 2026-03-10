package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== TerminalClient ====================

// ObserveTerminal gets terminal output from another pod.
func (c *GRPCCollaborationClient) ObserveTerminal(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*tools.TerminalOutput, error) {
	params := map[string]interface{}{
		"pod_key":        podKey,
		"lines":          lines,
		"raw":            raw,
		"include_screen": includeScreen,
	}
	var result tools.TerminalOutput
	if err := c.call(ctx, "observe_terminal", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendTerminalText sends text input to a terminal.
func (c *GRPCCollaborationClient) SendTerminalText(ctx context.Context, podKey string, text string) error {
	params := map[string]interface{}{
		"pod_key": podKey,
		"text":    text,
	}
	return c.call(ctx, "send_terminal_text", params, nil)
}

// SendTerminalKey sends special keys to a terminal.
func (c *GRPCCollaborationClient) SendTerminalKey(ctx context.Context, podKey string, keys []string) error {
	params := map[string]interface{}{
		"pod_key": podKey,
		"keys":    keys,
	}
	return c.call(ctx, "send_terminal_key", params, nil)
}
