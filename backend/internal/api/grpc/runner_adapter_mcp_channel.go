package grpc

import (
	"context"
	"strconv"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
)

// mcpChannelResponse wraps channel.Channel to add resolved slug fields for MCP responses.
type mcpChannelResponse struct {
	*channelDomain.Channel
	TicketSlug string `json:"ticket_slug,omitempty"`
}

// enrichChannelForMCP resolves the channel's ticket ID to its slug.
func (a *GRPCRunnerAdapter) enrichChannelForMCP(ctx context.Context, orgID int64, ch *channelDomain.Channel) *mcpChannelResponse {
	resp := &mcpChannelResponse{Channel: ch}
	if ch.TicketID != nil {
		t, err := a.ticketService.GetTicketByIDOrSlug(ctx, orgID, strconv.FormatInt(*ch.TicketID, 10))
		if err == nil {
			resp.TicketSlug = t.Slug
		}
	}
	return resp
}

// validateChannelAccess fetches a channel and checks organization-level access.
func (a *GRPCRunnerAdapter) validateChannelAccess(ctx context.Context, tc *middleware.TenantContext, channelID int64) (*channelDomain.Channel, *mcpError) {
	ch, err := a.channelService.GetChannel(ctx, channelID)
	if err != nil {
		return nil, newMcpError(404, "channel not found")
	}
	if ch.OrganizationID != tc.OrganizationID {
		return nil, newMcpError(403, "access denied")
	}
	return ch, nil
}

// mcpSearchChannels handles the "search_channels" MCP method.
func (a *GRPCRunnerAdapter) mcpSearchChannels(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		Name         string  `json:"name"`
		RepositoryID *int64  `json:"repository_id"`
		TicketSlug   *string `json:"ticket_slug"`
		IsArchived   *bool   `json:"is_archived"`
		Offset       int     `json:"offset"`
		Limit        int     `json:"limit"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}

	includeArchived := false
	if params.IsArchived != nil {
		includeArchived = *params.IsArchived
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	var ticketID *int64
	if params.TicketSlug != nil && *params.TicketSlug != "" {
		t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.TicketSlug)
		if err == nil {
			ticketID = &t.ID
		}
	}

	channels, _, mcpErr := a.channelService.ListChannels(ctx, tc.OrganizationID, &channelDomain.ChannelListFilter{
		IncludeArchived: includeArchived, RepositoryID: params.RepositoryID,
		TicketID: ticketID, Limit: limit, Offset: params.Offset,
	})
	if mcpErr != nil {
		return nil, newMcpError(500, "failed to list channels")
	}

	enriched := make([]*mcpChannelResponse, len(channels))
	for i, ch := range channels {
		enriched[i] = a.enrichChannelForMCP(ctx, tc.OrganizationID, ch)
	}
	return map[string]interface{}{"channels": enriched}, nil
}

// mcpCreateChannel handles the "create_channel" MCP method.
func (a *GRPCRunnerAdapter) mcpCreateChannel(ctx context.Context, tc *middleware.TenantContext, podKey string, payload []byte) (interface{}, *mcpError) {
	var params struct {
		Name         string  `json:"name"`
		Description  string  `json:"description"`
		RepositoryID *int64  `json:"repository_id"`
		TicketSlug   *string `json:"ticket_slug"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	if params.Name == "" {
		return nil, newMcpError(400, "name is required")
	}

	var desc *string
	if params.Description != "" {
		desc = &params.Description
	}
	var ticketID *int64
	if params.TicketSlug != nil && *params.TicketSlug != "" {
		t, err := a.ticketService.GetTicketByIDOrSlug(ctx, tc.OrganizationID, *params.TicketSlug)
		if err != nil {
			return nil, newMcpError(404, "ticket not found")
		}
		ticketID = &t.ID
	}

	ch, err := a.channelService.CreateChannel(ctx, &channel.CreateChannelRequest{
		OrganizationID: tc.OrganizationID, Name: params.Name, Description: desc,
		RepositoryID: params.RepositoryID, TicketID: ticketID,
		CreatedByPod: &podKey, CreatedByUserID: &tc.UserID,
	})
	if err != nil {
		if err == channel.ErrDuplicateName {
			return nil, newMcpError(409, "channel name already exists")
		}
		return nil, newMcpError(500, "failed to create channel")
	}
	_ = a.channelService.JoinChannel(ctx, ch.ID, podKey)
	return map[string]interface{}{"channel": a.enrichChannelForMCP(ctx, tc.OrganizationID, ch)}, nil
}

// mcpGetChannel handles the "get_channel" MCP method.
func (a *GRPCRunnerAdapter) mcpGetChannel(ctx context.Context, tc *middleware.TenantContext, payload []byte) (interface{}, *mcpError) {
	var params struct {
		ChannelID int64 `json:"channel_id"`
	}
	if err := unmarshalPayload(payload, &params); err != nil {
		return nil, err
	}
	if params.ChannelID == 0 {
		return nil, newMcpError(400, "channel_id is required")
	}

	ch, mcpErr := a.validateChannelAccess(ctx, tc, params.ChannelID)
	if mcpErr != nil {
		return nil, mcpErr
	}
	return map[string]interface{}{"channel": a.enrichChannelForMCP(ctx, tc.OrganizationID, ch)}, nil
}
