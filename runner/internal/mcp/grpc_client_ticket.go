package mcp

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// ==================== TicketClient ====================

// SearchTickets searches for tickets.
func (c *GRPCCollaborationClient) SearchTickets(ctx context.Context, repositoryID *int, status *tools.TicketStatus, priority *tools.TicketPriority, assigneeID *int, parentTicketSlug *string, query string, limit, page int) ([]tools.Ticket, error) {
	params := map[string]interface{}{
		"limit": limit,
		"page":  page,
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if status != nil {
		params["status"] = string(*status)
	}
	if priority != nil {
		params["priority"] = string(*priority)
	}
	if assigneeID != nil {
		params["assignee_id"] = *assigneeID
	}
	if parentTicketSlug != nil {
		params["parent_ticket_slug"] = *parentTicketSlug
	}
	if query != "" {
		params["query"] = query
	}
	var result struct {
		Tickets []tools.Ticket `json:"tickets"`
	}
	if err := c.call(ctx, "search_tickets", params, &result); err != nil {
		return nil, err
	}
	return result.Tickets, nil
}

// GetTicket gets a ticket by slug with optional content pagination.
func (c *GRPCCollaborationClient) GetTicket(ctx context.Context, ticketSlug string, contentOffset, contentLimit *int) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
	}
	if contentOffset != nil {
		params["content_offset"] = *contentOffset
	}
	if contentLimit != nil {
		params["content_limit"] = *contentLimit
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "get_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// CreateTicket creates a new ticket.
func (c *GRPCCollaborationClient) CreateTicket(ctx context.Context, repositoryID *int64, title, content string, priority tools.TicketPriority, parentTicketSlug *string) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"title":    title,
		"priority": priority,
	}
	if content != "" {
		params["content"] = content
	}
	if repositoryID != nil {
		params["repository_id"] = *repositoryID
	}
	if parentTicketSlug != nil {
		params["parent_ticket_slug"] = *parentTicketSlug
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "create_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// UpdateTicket updates a ticket.
func (c *GRPCCollaborationClient) UpdateTicket(ctx context.Context, ticketSlug string, title, content *string, status *tools.TicketStatus, priority *tools.TicketPriority) (*tools.Ticket, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
	}
	if title != nil {
		params["title"] = *title
	}
	if content != nil {
		params["content"] = *content
	}
	if status != nil {
		params["status"] = *status
	}
	if priority != nil {
		params["priority"] = *priority
	}
	var result struct {
		Ticket tools.Ticket `json:"ticket"`
	}
	if err := c.call(ctx, "update_ticket", params, &result); err != nil {
		return nil, err
	}
	return &result.Ticket, nil
}

// PostComment posts a comment on a ticket.
func (c *GRPCCollaborationClient) PostComment(ctx context.Context, ticketSlug, content string, parentID *int64) (*tools.TicketComment, error) {
	params := map[string]interface{}{
		"ticket_slug": ticketSlug,
		"content":     content,
	}
	if parentID != nil {
		params["parent_id"] = *parentID
	}
	var result struct {
		Comment tools.TicketComment `json:"comment"`
	}
	if err := c.call(ctx, "post_comment", params, &result); err != nil {
		return nil, err
	}
	return &result.Comment, nil
}

// ==================== PodClient ====================

// CreatePod creates a new AgentPod.
func (c *GRPCCollaborationClient) CreatePod(ctx context.Context, req *tools.PodCreateRequest) (*tools.PodCreateResponse, error) {
	var result struct {
		Pod struct {
			PodKey string `json:"pod_key"`
			Status string `json:"status"`
		} `json:"pod"`
	}
	if err := c.call(ctx, "create_pod", req, &result); err != nil {
		return nil, err
	}
	return &tools.PodCreateResponse{
		PodKey: result.Pod.PodKey,
		Status: result.Pod.Status,
	}, nil
}
