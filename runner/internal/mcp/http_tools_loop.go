package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Loop Tools

func (s *HTTPServer) createListLoopsTool() *MCPTool {
	return &MCPTool{
		Name:        "list_loops",
		Description: "List automated loops in the organization. Loops are repeatable tasks that can be triggered manually or via cron.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"enabled", "disabled", "archived"},
					"description": "Filter by loop status",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for loop name",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 20)",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Pagination offset (default: 0)",
				},
			},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			status := getStringArg(args, "status")
			query := getStringArg(args, "query")

			limit := getIntArg(args, "limit")
			if limit == 0 {
				limit = 20
			}
			offset := getIntArg(args, "offset")

			result, err := client.ListLoops(ctx, status, query, limit, offset)
			if err != nil {
				return nil, err
			}
			return tools.LoopSummaryList(result), nil
		},
	}
}

func (s *HTTPServer) createTriggerLoopTool() *MCPTool {
	return &MCPTool{
		Name:        "trigger_loop",
		Description: "Manually trigger a loop run. Optionally pass runtime variables to override the loop's default prompt variables.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"loop_slug": map[string]interface{}{
					"type":        "string",
					"description": "The slug of the loop to trigger. Use list_loops to find available loops.",
				},
				"variables": map[string]interface{}{
					"type":        "object",
					"description": "Runtime variables to override prompt template placeholders (optional)",
				},
			},
			"required": []string{"loop_slug"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			loopSlug := getStringArg(args, "loop_slug")
			if loopSlug == "" {
				return nil, fmt.Errorf("loop_slug is required")
			}
			variables := getMapArg(args, "variables")

			return client.TriggerLoop(ctx, loopSlug, variables)
		},
	}
}
