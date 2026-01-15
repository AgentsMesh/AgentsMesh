package mcp

import (
	"context"

	"github.com/anthropics/agentmesh/runner/internal/mcp/tools"
)

// Discovery Tools

func (s *HTTPServer) createListAvailablePodsTool() *MCPTool {
	return &MCPTool{
		Name:        "list_available_pods",
		Description: "List other agent pods available for collaboration. Shows pods that can be bound to.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.ListAvailablePods(ctx)
		},
	}
}

func (s *HTTPServer) createListRunnersTool() *MCPTool {
	return &MCPTool{
		Name:        "list_runners",
		Description: "List available runners in the organization. Shows runner status, capacity, and host information.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.ListRunners(ctx)
		},
	}
}

func (s *HTTPServer) createListRepositoriesTool() *MCPTool {
	return &MCPTool{
		Name:        "list_repositories",
		Description: "List repositories configured in the organization. Shows repository name, provider, clone URL, and default branch.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			return client.ListRepositories(ctx)
		},
	}
}
