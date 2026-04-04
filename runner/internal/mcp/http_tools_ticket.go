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
					"enum":        []string{"backlog", "todo", "in_progress", "in_review", "done"},
					"description": "Filter by ticket status",
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
				"parent_ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Filter by parent ticket slug (e.g., 'AM-123') for subtasks",
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
			parentTicketSlug := getStringPtrArg(args, "parent_ticket_slug")
			query := getStringArg(args, "query")

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

			limit := getIntArg(args, "limit")
			if limit == 0 {
				limit = 20
			}

			page := getIntArg(args, "page")
			if page == 0 {
				page = 1
			}

			result, err := client.SearchTickets(ctx, repositoryID, status, priority, assigneeID, parentTicketSlug, query, limit, page)
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
		Description: "Get details of a specific ticket by its slug (e.g., 'AM-123').",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Ticket slug (e.g., 'AM-123'). Use search_tickets to find available tickets.",
				},
				"content_offset": map[string]interface{}{
					"type":        "integer",
					"description": "Start line number (0-based) for reading ticket content. Default is 0.",
				},
				"content_limit": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to read from ticket content. Default is 200.",
				},
			},
			"required": []string{"ticket_slug"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			ticketSlug := getStringArg(args, "ticket_slug")
			if ticketSlug == "" {
				return nil, fmt.Errorf("ticket_slug is required")
			}
			contentOffset := getIntPtrArg(args, "content_offset")
			contentLimit := getIntPtrArg(args, "content_limit")
			return client.GetTicket(ctx, ticketSlug, contentOffset, contentLimit)
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
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content of the ticket (optional)",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"urgent", "high", "medium", "low", "none"},
					"description": "Priority level (default: medium)",
				},
				"parent_ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Parent ticket slug (e.g., 'AM-123') for creating subtasks",
				},
			},
			"required": []string{"title"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			repositoryID := getInt64PtrArg(args, "repository_id")
			title := getStringArg(args, "title")
			priority := getStringArg(args, "priority")
			parentTicketSlug := getStringPtrArg(args, "parent_ticket_slug")

			if title == "" {
				return nil, fmt.Errorf("title is required")
			}

			if priority == "" {
				priority = "medium"
			}

			content := getStringArg(args, "content")
			return client.CreateTicket(ctx, repositoryID, title, content, tools.TicketPriority(priority), parentTicketSlug)
		},
	}
}

func (s *HTTPServer) createPostCommentTool() *MCPTool {
	return &MCPTool{
		Name:        "post_comment",
		Description: "Post a comment on a ticket. Optionally reply to an existing comment by providing parent_id.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Ticket slug (e.g., 'AM-123')",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Comment content",
				},
				"parent_id": map[string]interface{}{
					"type":        "integer",
					"description": "ID of the parent comment to reply to (optional)",
				},
			},
			"required": []string{"ticket_slug", "content"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			ticketSlug := getStringArg(args, "ticket_slug")
			if ticketSlug == "" {
				return nil, fmt.Errorf("ticket_slug is required")
			}
			content := getStringArg(args, "content")
			if content == "" {
				return nil, fmt.Errorf("content is required")
			}
			parentID := getInt64PtrArg(args, "parent_id")
			return client.PostComment(ctx, ticketSlug, content, parentID)
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
				"ticket_slug": map[string]interface{}{
					"type":        "string",
					"description": "Ticket slug to update (e.g., 'AM-123')",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "New title (optional)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "New content (optional)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"backlog", "todo", "in_progress", "in_review", "done"},
					"description": "New status (optional)",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"urgent", "high", "medium", "low", "none"},
					"description": "New priority (optional)",
				},
			},
			"required": []string{"ticket_slug"},
		},
		Handler: func(ctx context.Context, client tools.CollaborationClient, args map[string]interface{}) (interface{}, error) {
			ticketSlug := getStringArg(args, "ticket_slug")
			if ticketSlug == "" {
				return nil, fmt.Errorf("ticket_slug is required")
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

			var content *string
			if d := getStringArg(args, "content"); d != "" {
				content = &d
			}

			return client.UpdateTicket(ctx, ticketSlug, title, content, status, priority)
		},
	}
}
