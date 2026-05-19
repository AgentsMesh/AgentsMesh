package grpc

import (
	"context"
	"strconv"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	ticketDomain "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/anthropics/agentsmesh/backend/pkg/blocknote"
)

type mcpTicketResponse struct {
	*ticketDomain.Ticket
	ParentTicketSlug  string `json:"parent_ticket_slug,omitempty"`
	ContentTotalLines int    `json:"content_total_lines,omitempty"` // Total lines after conversion
	ContentOffset     int    `json:"content_offset,omitempty"`      // Start line of this response (0-based)
	ContentLimit      int    `json:"content_limit,omitempty"`       // Number of lines returned
}

type contentPaginationParams struct {
	Offset int
	Limit  int
}

func (a *GRPCRunnerAdapter) enrichTicketForMCP(ctx context.Context, orgID int64, t *ticketDomain.Ticket, pagination *contentPaginationParams) *mcpTicketResponse {
	ticketCopy := *t
	resp := &mcpTicketResponse{Ticket: &ticketCopy}

	if t.ParentTicketID != nil {
		parent, err := a.ticketService.GetTicketByIDOrSlug(ctx, orgID, strconv.FormatInt(*t.ParentTicketID, 10))
		if err == nil {
			resp.ParentTicketSlug = parent.Slug
		}
	}

	plainText := ""
	switch {
	case ticketCopy.Content != nil && *ticketCopy.Content != "":
		plainText = blocknote.ToPlainText(*ticketCopy.Content)
	case t.ContentBlockID != nil && a.blockstoreService != nil:
		actor := blockstoreservice.ActorContext{
			OrgID: orgID, UserID: t.ReporterID,
			ActorType: blockstore.ActorUser, ActorID: t.ReporterID,
		}
		block, err := a.blockstoreService.GetBlock(ctx, actor, *t.ContentBlockID)
		if err == nil && block != nil && block.Text != nil {
			plainText = *block.Text
		}
	}

	if plainText != "" {
		lines := strings.Split(plainText, "\n")
		totalLines := len(lines)

		offset := 0
		limit := 200
		if pagination != nil {
			offset = pagination.Offset
			limit = pagination.Limit
		}

		if offset < 0 {
			offset = 0
		}
		if offset > totalLines {
			offset = totalLines
		}

		end := offset + limit
		if end > totalLines {
			end = totalLines
		}

		selected := lines[offset:end]
		content := strings.Join(selected, "\n")
		ticketCopy.Content = &content

		resp.ContentTotalLines = totalLines
		resp.ContentOffset = offset
		resp.ContentLimit = len(selected)
	}

	return resp
}

func (a *GRPCRunnerAdapter) mcpSearchTickets(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		RepositoryID      *int64  `json:"repository_id"`
		Status            string  `json:"status"`
		Priority          string  `json:"priority"`
		AssigneeID        *int64  `json:"assignee_id"`
		ParentTicketSlug  *string `json:"parent_ticket_slug"`
		Query             string  `json:"query"`
		Limit             int     `json:"limit"`
		Page              int     `json:"page"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	offset := 0
	if params.Page > 0 {
		offset = (params.Page - 1) * limit
	}

	var parentTicketID *int64
	if params.ParentTicketSlug != nil && *params.ParentTicketSlug != "" {
		parentTicket, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.ParentTicketSlug)
		if err != nil {
			return nil, newMcpError(404, "parent ticket not found")
		}
		parentTicketID = &parentTicket.ID
	}

	tickets, _, err := a.ticketService.ListTickets(ctx, &ticketDomain.TicketListFilter{
		OrganizationID: tc.OrganizationID,
		RepositoryID:   params.RepositoryID,
		Status:         params.Status,
		Priority:       params.Priority,
		AssigneeID:     params.AssigneeID,
		ParentTicketID: parentTicketID,
		Query:          params.Query,
		UserRole:       tc.UserRole,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, newMcpError(500, "failed to search tickets")
	}

	enriched := make([]*mcpTicketResponse, len(tickets))
	for i, t := range tickets {
		enriched[i] = a.enrichTicketForMCP(ctx, tc.OrganizationID, t, nil)
	}
	return map[string]interface{}{"tickets": enriched}, nil
}

func (a *GRPCRunnerAdapter) mcpGetTicket(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		TicketSlug    string `json:"ticket_slug"`
		ContentOffset *int   `json:"content_offset"` // Start line (0-based), default 0
		ContentLimit  *int   `json:"content_limit"`  // Number of lines, default 200
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

	pagination := &contentPaginationParams{Offset: 0, Limit: 200}
	if params.ContentOffset != nil {
		pagination.Offset = *params.ContentOffset
	}
	if params.ContentLimit != nil {
		pagination.Limit = *params.ContentLimit
	}

	return map[string]interface{}{"ticket": a.enrichTicketForMCP(ctx, tc.OrganizationID, t, pagination)}, nil
}
