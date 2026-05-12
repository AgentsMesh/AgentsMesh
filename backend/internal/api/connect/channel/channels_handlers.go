package channelconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	channeldomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelservice "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

// defaultChannelListLimit mirrors REST's hard-coded 50 (channels.go:65).
const defaultChannelListLimit = 50

func (s *Server) ListChannels(
	ctx context.Context, req *connect.Request[channelv1.ListChannelsRequest],
) (*connect.Response[channelv1.ListChannelsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	filter := &channeldomain.ChannelListFilter{
		IncludeArchived: req.Msg.GetIncludeArchived(),
		Limit:           int(defaultListLimit(req.Msg.Limit, defaultChannelListLimit)),
		Offset:          int(defaultListOffset(req.Msg.Offset)),
	}
	if req.Msg.RepositoryId != nil {
		v := req.Msg.GetRepositoryId()
		filter.RepositoryID = &v
	}
	if slug := req.Msg.GetTicketSlug(); slug != "" {
		t, err := s.ticketSvc.GetTicketByIDOrSlug(ctx, tenant.OrganizationID, slug)
		if err == nil {
			filter.TicketID = &t.ID
		} else {
			// REST returns empty results when the ticket slug doesn't
			// resolve (channels.go:53). Mirror that here.
			return connect.NewResponse(&channelv1.ListChannelsResponse{
				Items: nil, Total: 0,
				Limit:  int32(filter.Limit),
				Offset: int32(filter.Offset),
			}), nil
		}
	}

	channels, total, err := s.channelSvc.ListChannels(ctx, tenant.OrganizationID, tenant.UserID, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*channelv1.Channel, 0, len(channels))
	for _, c := range channels {
		items = append(items, toProtoChannel(c))
	}
	return connect.NewResponse(&channelv1.ListChannelsResponse{
		Items:  items,
		Total:  total,
		Limit:  int32(filter.Limit),
		Offset: int32(filter.Offset),
	}), nil
}

func (s *Server) GetChannel(
	ctx context.Context, req *connect.Request[channelv1.GetChannelRequest],
) (*connect.Response[channelv1.Channel], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	enriched, err := s.channelSvc.GetChannelForUser(ctx, ch.ID, tenant.UserID)
	if err != nil {
		// requireChannelAccess already validated existence; fall back to ch.
		return connect.NewResponse(toProtoChannel(ch)), nil
	}
	return connect.NewResponse(toProtoChannel(enriched)), nil
}

func (s *Server) CreateChannel(
	ctx context.Context, req *connect.Request[channelv1.CreateChannelRequest],
) (*connect.Response[channelv1.Channel], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	tenant := middleware.GetTenant(ctx)
	createReq := &channelservice.CreateChannelRequest{
		OrganizationID:   tenant.OrganizationID,
		Name:             req.Msg.GetName(),
		Description:      optionalString(req.Msg.Description),
		RepositoryID:     nilIfZeroPtr(req.Msg.RepositoryId),
		CreatedByUserID:  &tenant.UserID,
		Visibility:       req.Msg.GetVisibility(),
		InitialMemberIDs: req.Msg.GetMemberIds(),
	}
	if slug := req.Msg.GetTicketSlug(); slug != "" {
		t, err := s.ticketSvc.GetTicketByIDOrSlug(ctx, tenant.OrganizationID, slug)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("ticket not found"))
		}
		createReq.TicketID = &t.ID
	}

	ch, err := s.channelSvc.CreateChannel(ctx, createReq)
	if err != nil {
		return nil, mapServiceError(err)
	}
	enriched, err := s.channelSvc.GetChannelForUser(ctx, ch.ID, tenant.UserID)
	if err != nil {
		return connect.NewResponse(toProtoChannel(ch)), nil
	}
	return connect.NewResponse(toProtoChannel(enriched)), nil
}

func (s *Server) UpdateChannel(
	ctx context.Context, req *connect.Request[channelv1.UpdateChannelRequest],
) (*connect.Response[channelv1.Channel], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	updated, err := s.channelSvc.UpdateChannel(
		ctx, ch.ID,
		optionalString(req.Msg.Name),
		optionalString(req.Msg.Description),
		optionalString(req.Msg.Document),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	tenant := middleware.GetTenant(ctx)
	enriched, err := s.channelSvc.GetChannelForUser(ctx, updated.ID, tenant.UserID)
	if err != nil {
		return connect.NewResponse(toProtoChannel(updated)), nil
	}
	return connect.NewResponse(toProtoChannel(enriched)), nil
}

func (s *Server) ArchiveChannel(
	ctx context.Context, req *connect.Request[channelv1.ArchiveChannelRequest],
) (*connect.Response[channelv1.ArchiveChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.channelSvc.ArchiveChannel(ctx, ch.ID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.ArchiveChannelResponse{Message: "Channel archived"}), nil
}

func (s *Server) UnarchiveChannel(
	ctx context.Context, req *connect.Request[channelv1.UnarchiveChannelRequest],
) (*connect.Response[channelv1.UnarchiveChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.channelSvc.UnarchiveChannel(ctx, ch.ID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.UnarchiveChannelResponse{Message: "Channel unarchived"}), nil
}

// nilIfZeroPtr treats a proto-optional int64 of 0 as "absent" because the
// REST handler exposed RepositoryID as a nullable pointer; passing 0 would
// mean "match repository_id = 0" which is never desired.
func nilIfZeroPtr(p *int64) *int64 {
	if p == nil || *p == 0 {
		return nil
	}
	v := *p
	return &v
}
