package mcp

import (
	"context"
	"encoding/json"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== LoopClient ====================

// ListLoops lists loops for the pod's organization.
func (c *GRPCCollaborationClient) ListLoops(ctx context.Context, status, query string, limit, offset int) ([]tools.LoopSummary, error) {
	params := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}
	if status != "" {
		params["status"] = status
	}
	if query != "" {
		params["query"] = query
	}
	var result struct {
		Loops []tools.LoopSummary `json:"loops"`
	}
	if err := c.call(ctx, "list_loops", params, &result); err != nil {
		return nil, err
	}
	return result.Loops, nil
}

// TriggerLoop triggers a loop run by slug.
func (c *GRPCCollaborationClient) TriggerLoop(ctx context.Context, loopSlug string, variables map[string]interface{}) (*tools.LoopTriggerResult, error) {
	params := map[string]interface{}{
		"loop_slug": loopSlug,
	}
	if len(variables) > 0 {
		varsJSON, err := json.Marshal(variables)
		if err != nil {
			return nil, err
		}
		params["variables"] = json.RawMessage(varsJSON)
	}

	var result tools.LoopTriggerResult
	if err := c.call(ctx, "trigger_loop", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
