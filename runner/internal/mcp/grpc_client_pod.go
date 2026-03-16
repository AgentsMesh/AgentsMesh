package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== PodInteractionClient ====================

// GetPodSnapshot gets pod output from another pod.
func (c *GRPCCollaborationClient) GetPodSnapshot(ctx context.Context, podKey string, lines int, raw bool, includeScreen bool) (*tools.PodSnapshot, error) {
	params := map[string]interface{}{
		"pod_key":        podKey,
		"lines":          lines,
		"raw":            raw,
		"include_screen": includeScreen,
	}
	var result tools.PodSnapshot
	if err := c.call(ctx, "get_pod_snapshot", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendPodInput sends text and/or special keys to a pod.
func (c *GRPCCollaborationClient) SendPodInput(ctx context.Context, podKey string, text string, keys []string) error {
	params := map[string]interface{}{
		"pod_key": podKey,
		"text":    text,
		"keys":    keys,
	}
	return c.call(ctx, "send_pod_input", params, nil)
}
