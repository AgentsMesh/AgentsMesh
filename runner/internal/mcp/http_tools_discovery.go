package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
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
			result, err := client.ListAvailablePods(ctx)
			if err != nil {
				return nil, err
			}
			return tools.AvailablePodList(result), nil
		},
	}
}

func (s *HTTPServer) createListRunnersTool() *MCPTool {
	return &MCPTool{
		Name:        "list_runners",
		Description: "List available runners with their supported agents. Returns runner ID, status, capacity, and available_agents array containing full agent details (id, slug, name, description, config schema, user_config). Use the agent slug when creating pods with create_pod tool.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			result, err := client.ListRunners(ctx)
			if err != nil {
				return nil, err
			}
			return tools.RunnerSummaryList(result), nil
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
			result, err := client.ListRepositories(ctx)
			if err != nil {
				return nil, err
			}
			return tools.RepositoryList(result), nil
		},
	}
}
