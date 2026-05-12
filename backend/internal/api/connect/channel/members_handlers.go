package channelconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

const (
	defaultMembersLimit = 50
	maxMembersLimit     = 200
)

func (s *Server) ListChannelMembers(
	ctx context.Context, req *connect.Request[channelv1.ListChannelMembersRequest],
) (*connect.Response[channelv1.ListChannelMembersResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	limit := clampLimit(req.Msg.Limit, defaultMembersLimit, maxMembersLimit)
	offset := defaultListOffset(req.Msg.Offset)

	members, total, err := s.channelSvc.ListMembers(ctx, ch.ID, int(limit), int(offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*channelv1.ChannelMember, 0, len(members))
	for _, m := range members {
		items = append(items, toProtoMember(m))
	}
	return connect.NewResponse(&channelv1.ListChannelMembersResponse{
		Items: items, Total: total, Limit: limit, Offset: offset,
	}), nil
}

func (s *Server) JoinChannel(
	ctx context.Context, req *connect.Request[channelv1.JoinChannelRequest],
) (*connect.Response[channelv1.JoinChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.JoinPublicChannel(ctx, ch.ID, tenant.UserID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.JoinChannelResponse{Message: "joined"}), nil
}

func (s *Server) LeaveChannel(
	ctx context.Context, req *connect.Request[channelv1.LeaveChannelRequest],
) (*connect.Response[channelv1.LeaveChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.LeaveUserChannel(ctx, ch.ID, tenant.UserID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.LeaveChannelResponse{Message: "left"}), nil
}

func (s *Server) InviteChannelMembers(
	ctx context.Context, req *connect.Request[channelv1.InviteChannelMembersRequest],
) (*connect.Response[channelv1.InviteChannelMembersResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.InviteMembers(ctx, ch.ID, tenant.UserID, req.Msg.GetUserIds()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.InviteChannelMembersResponse{Message: "invited"}), nil
}

func (s *Server) RemoveChannelMember(
	ctx context.Context, req *connect.Request[channelv1.RemoveChannelMemberRequest],
) (*connect.Response[channelv1.RemoveChannelMemberResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.RemoveMember(ctx, ch.ID, tenant.UserID, req.Msg.GetUserId()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.RemoveChannelMemberResponse{Message: "removed"}), nil
}
