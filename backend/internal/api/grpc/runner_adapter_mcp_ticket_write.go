package grpc

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// mcpCreateTicket handles the "create_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpCreateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID     *int64  `json:"repository_id"`
		Title            string  `json:"title"`
		Content          string  `json:"content"`
		Priority         string  `json:"priority"`
		ParentTicketSlug *string `json:"parent_ticket_slug"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.Title == "" {
		return nil, newMcpError(400, "title is required")
	}
	if params.Priority == "" {
		params.Priority = "medium"
	}

	var content *string
	if params.Content != "" {
		content = &params.Content
	}

	// Resolve parent ticket slug to ID
	var parentTicketID *int64
	if params.ParentTicketSlug != nil && *params.ParentTicketSlug != "" {
		parentTicket, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.ParentTicketSlug)
		if err != nil {
			return nil, newMcpError(404, "parent ticket not found")
		}
		parentTicketID = &parentTicket.ID
	}

	t, err := a.ticketService.CreateTicket(ctx, &ticket.CreateTicketRequest{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		ReporterID:     tc.UserID,
		Title:          params.Title,
		Content:        content,
		Priority:       params.Priority,
		ParentTicketID: parentTicketID,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to create ticket")
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)}, nil
}

// mcpUpdateTicket handles the "update_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpUpdateTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug string  `json:"ticket_slug"`
		Title      *string `json:"title"`
		Content    *string `json:"content"`
		Status     *string `json:"status"`
		Priority   *string `json:"priority"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	updates := make(map[string]interface{})
	if params.Title != nil {
		updates["title"] = *params.Title
	}
	if params.Content != nil {
		updates["content"] = *params.Content
	}
	if params.Status != nil {
		updates["status"] = *params.Status
	}
	if params.Priority != nil {
		updates["priority"] = *params.Priority
	}

	t, err = a.ticketService.UpdateTicket(ctx, t.ID, updates)
	if err != nil {
		return nil, newMcpError(500, "failed to update ticket")
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)}, nil
}

// mcpPostComment handles the "post_comment" MCP method.
func (a *GRPCRunnerAdapter) mcpPostComment(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug string `json:"ticket_slug"`
		Content    string `json:"content"`
		ParentID   *int64 `json:"parent_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}
	if params.Content == "" {
		return nil, newMcpError(400, "content is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	comment, err := a.ticketService.CreateComment(ctx, t.ID, tc.UserID, params.Content, params.ParentID, nil)
	if err != nil {
		return nil, newMcpError(500, "failed to post comment")
	}

	return map[string]interface{}{"comment": comment}, nil
}

// mcpDeleteTicket handles the "delete_ticket" MCP method.
func (a *GRPCRunnerAdapter) mcpDeleteTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug string `json:"ticket_slug"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	if params.TicketSlug == "" {
		return nil, newMcpError(400, "ticket_slug is required")
	}

	t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, params.TicketSlug)
	if err != nil {
		return nil, newMcpError(404, "ticket not found")
	}

	if err := a.ticketService.DeleteTicket(ctx, t.ID); err != nil {
		return nil, newMcpError(500, "failed to delete ticket")
	}

	return map[string]interface{}{"message": "Ticket deleted"}, nil
}
