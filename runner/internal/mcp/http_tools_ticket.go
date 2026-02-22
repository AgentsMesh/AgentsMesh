package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Ticket Tools

func (s *HTTPServer) createSearchTicketsTool() *MCPTool {
	return &MCPTool{
		Name:        "search_tickets",
		Description: "Search for tickets/tasks in the project management system.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by repository ID. Use list_repositories to see available repositories.",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"backlog", "todo", "in_progress", "in_review", "done", "canceled"},
					"description": "Filter by ticket status",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"task", "bug", "feature", "improvement", "epic", "subtask", "story"},
					"description": "Filter by ticket type",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"urgent", "high", "medium", "low", "none"},
					"description": "Filter by priority",
				},
				"assignee_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by assignee user ID",
				},
				"parent_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by parent ticket ID (for subtasks)",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for title",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 20)",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "Page number (default: 1)",
				},
			},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			repositoryID := getIntPtrArg(args, "repository_id")
			assigneeID := getIntPtrArg(args, "assignee_id")
			parentID := getIntPtrArg(args, "parent_id")
			query := getStringArg(args, "query")

			var status *tools.TicketStatus
			if s := getStringArg(args, "status"); s != "" {
				ts := tools.TicketStatus(s)
				status = &ts
			}

			var ticketType *tools.TicketType
			if t := getStringArg(args, "type"); t != "" {
				tt := tools.TicketType(t)
				ticketType = &tt
			}

			var priority *tools.TicketPriority
			if p := getStringArg(args, "priority"); p != "" {
				tp := tools.TicketPriority(p)
				priority = &tp
			}

			limit := getIntArg(args, "limit")
			if limit == 0 {
				limit = 20
			}

			page := getIntArg(args, "page")
			if page == 0 {
				page = 1
			}

			result, err := client.SearchTickets(ctx, repositoryID, status, ticketType, priority, assigneeID, parentID, query, limit, page)
			if err != nil {
				return nil, err
			}
			return tools.TicketList(result), nil
		},
	}
}

func (s *HTTPServer) createGetTicketTool() *MCPTool {
	return &MCPTool{
		Name:        "get_ticket",
		Description: "Get details of a specific ticket by ID or identifier (e.g., 'AM-123').",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ticket_id": map[string]interface{}{
					"type":        "string",
					"description": "Ticket ID (numeric) or identifier (e.g., 'AM-123')",
				},
			},
			"required": []string{"ticket_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			ticketID := getStringArg(args, "ticket_id")
			if ticketID == "" {
				return nil, fmt.Errorf("ticket_id is required")
			}
			return client.GetTicket(ctx, ticketID)
		},
	}
}

func (s *HTTPServer) createCreateTicketTool() *MCPTool {
	return &MCPTool{
		Name:        "create_ticket",
		Description: "Create a new ticket/task in the project management system.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository_id": map[string]interface{}{
					"type":        "integer",
					"description": "The repository ID to associate the ticket with (optional). Use list_repositories to see available repositories.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title of the ticket",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"task", "bug", "feature", "improvement", "epic", "subtask", "story"},
					"description": "Type of ticket (default: task)",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"urgent", "high", "medium", "low", "none"},
					"description": "Priority level (default: medium)",
				},
				"parent_ticket_id": map[string]interface{}{
					"type":        "integer",
					"description": "Parent ticket ID for creating subtasks",
				},
			},
			"required": []string{"title"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			repositoryID := getInt64PtrArg(args, "repository_id")
			title := getStringArg(args, "title")
			ticketType := getStringArg(args, "type")
			priority := getStringArg(args, "priority")
			parentTicketID := getInt64PtrArg(args, "parent_ticket_id")

			if title == "" {
				return nil, fmt.Errorf("title is required")
			}

			if ticketType == "" {
				ticketType = "task"
			}
			if priority == "" {
				priority = "medium"
			}

			return client.CreateTicket(ctx, repositoryID, title, tools.TicketType(ticketType), tools.TicketPriority(priority), parentTicketID)
		},
	}
}

func (s *HTTPServer) createUpdateTicketTool() *MCPTool {
	return &MCPTool{
		Name:        "update_ticket",
		Description: "Update an existing ticket's fields.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ticket_id": map[string]interface{}{
					"type":        "string",
					"description": "Ticket ID or identifier to update",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "New title (optional)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"backlog", "todo", "in_progress", "in_review", "done", "canceled"},
					"description": "New status (optional)",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"urgent", "high", "medium", "low", "none"},
					"description": "New priority (optional)",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"task", "bug", "feature", "improvement", "epic", "subtask", "story"},
					"description": "New type (optional)",
				},
			},
			"required": []string{"ticket_id"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			ticketID := getStringArg(args, "ticket_id")
			if ticketID == "" {
				return nil, fmt.Errorf("ticket_id is required")
			}

			var title *string
			if v := getStringArg(args, "title"); v != "" {
				title = &v
			}

			var status *tools.TicketStatus
			if s := getStringArg(args, "status"); s != "" {
				ts := tools.TicketStatus(s)
				status = &ts
			}

			var priority *tools.TicketPriority
			if p := getStringArg(args, "priority"); p != "" {
				tp := tools.TicketPriority(p)
				priority = &tp
			}

			var ticketType *tools.TicketType
			if t := getStringArg(args, "type"); t != "" {
				tt := tools.TicketType(t)
				ticketType = &tt
			}

			return client.UpdateTicket(ctx, ticketID, title, status, priority, ticketType)
		},
	}
}
