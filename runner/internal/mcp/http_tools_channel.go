package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Channel Management Tools: search, create, get

func (s *HTTPServer) createSearchChannelsTool() *MCPTool {
	return &MCPTool{
		Name:        "search_channels",
		Description: "Search for collaboration channels. Channels are shared spaces for multi-agent communication.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Filter by channel name (partial match)",
				},
				"repository_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by repository ID. Use list_repositories to see available repositories.",
				},
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Filter by ticket slug (e.g., 'AM-123')",
				},
				"is_archived": map[string]interface{}{
					"type":        "boolean",
					"description": "Filter by archived status",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Pagination offset (default: 0)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results to return (default: 20)",
				},
			},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			name := getStringArg(args, "name")
			repositoryID := getIntPtrArg(args, "repository_id")
			ticketSlug := getStringPtrArg(args, "ticket_slug")

			var isArchived *bool
			if v, ok := args["is_archived"].(bool); ok {
				isArchived = &v
			}

			offset := getIntArg(args, "offset")
			limit := getIntArg(args, "limit")
			if limit == 0 {
				limit = 20
			}

			result, err := client.SearchChannels(ctx, name, repositoryID, ticketSlug, isArchived, offset, limit)
			if err != nil {
				return nil, err
			}
			return tools.ChannelList(result), nil
		},
	}
}

func (s *HTTPServer) createCreateChannelTool() *MCPTool {
	return &MCPTool{
		Name:        "create_channel",
		Description: "Create a new collaboration channel for multi-agent communication.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Unique name for the channel",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Description of the channel's purpose",
				},
				"repository_id": map[string]interface{}{
					"type":        "integer",
					"description": "Associated repository ID (optional). Use list_repositories to see available repositories.",
				},
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Associated ticket slug (e.g., 'AM-123', optional)",
				},
			},
			"required": []string{"name"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			name := getStringArg(args, "name")
			description := getStringArg(args, "description")
			repositoryID := getIntPtrArg(args, "repository_id")
			ticketSlug := getStringPtrArg(args, "ticket_slug")

			if name == "" {
				return nil, fmt.Errorf("name is required")
			}

			return client.CreateChannel(ctx, name, description, repositoryID, ticketSlug)
		},
	}
}

func (s *HTTPServer) createGetChannelTool() *MCPTool {
	return &MCPTool{
		Name:        "get_channel",
		Description: "Get details of a specific collaboration channel.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"channel_id": map[string]interface{}{
					"type":        "integer",
					"description": "The ID of the channel to retrieve",
				},
			},
			"required": []string{"channel_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			channelID := getIntArg(args, "channel_id")
			if channelID == 0 {
				return nil, fmt.Errorf("channel_id is required")
			}
			return client.GetChannel(ctx, channelID)
		},
	}
}
