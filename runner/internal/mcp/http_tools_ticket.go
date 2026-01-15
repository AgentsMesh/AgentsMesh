package mcp

import (
	"context"
	"fmt"

	"github.com/anthropics/agentmesh/runner/internal/mcp/tools"
)

// Ticket Tools

func (s *HTTPServer) createSearchTicketsTool() *MCPTool {
	return &MCPTool{
		Name:        "search_tickets",
		Description: "Search for tickets/tasks in the project management system.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"product_id": map[string]interface{}{
					"type":        "integer",
					"description": "Filter by product ID",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"backlog", "todo", "in_progress", "in_review", "done", "canceled"},
					"description": "Filter by ticket status",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"task", "bug", "feature", "improvement", "epic"},
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
					"description": "Search query for title/description",
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
			productID := getIntPtrArg(args, "product_id")
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

			return client.SearchTickets(ctx, productID, status, ticketType, priority, assigneeID, parentID, query, limit, page)
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
				"product_id": map[string]interface{}{
					"type":        "integer",
					"description": "The product ID to create the ticket in",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title of the ticket",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Detailed description of the ticket",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"task", "bug", "feature", "improvement", "epic"},
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
			"required": []string{"product_id", "title"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			productID := getIntArg(args, "product_id")
			title := getStringArg(args, "title")
			description := getStringArg(args, "description")
			ticketType := getStringArg(args, "type")
			priority := getStringArg(args, "priority")
			parentTicketID := getIntPtrArg(args, "parent_ticket_id")

			if productID == 0 || title == "" {
				return nil, fmt.Errorf("product_id and title are required")
			}

			if ticketType == "" {
				ticketType = "task"
			}
			if priority == "" {
				priority = "medium"
			}

			return client.CreateTicket(ctx, productID, title, description, tools.TicketType(ticketType), tools.TicketPriority(priority), parentTicketID)
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
				"description": map[string]interface{}{
					"type":        "string",
					"description": "New description (optional)",
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
					"enum":        []string{"task", "bug", "feature", "improvement", "epic"},
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

			var title, description *string
			if v := getStringArg(args, "title"); v != "" {
				title = &v
			}
			if v := getStringArg(args, "description"); v != "" {
				description = &v
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

			return client.UpdateTicket(ctx, ticketID, title, description, status, priority, ticketType)
		},
	}
}
