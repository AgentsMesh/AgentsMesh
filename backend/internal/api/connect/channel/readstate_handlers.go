package channelconnect

import (
	"context"
	"strconv"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

func (s *Server) MarkChannelRead(
	ctx context.Context, req *connect.Request[channelv1.MarkChannelReadRequest],
) (*connect.Response[channelv1.MarkChannelReadResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.MarkRead(ctx, ch.ID, tenant.UserID, req.Msg.GetMessageId()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.MarkChannelReadResponse{Status: "ok"}), nil
}

// GetChannelUnreadCounts is org-scoped at request-level but reads unread
// counts across all channels the caller is a member of in any org.
func (s *Server) GetChannelUnreadCounts(
	ctx context.Context, req *connect.Request[channelv1.GetChannelUnreadCountsRequest],
) (*connect.Response[channelv1.GetChannelUnreadCountsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	counts, err := s.channelSvc.GetUnreadCounts(ctx, tenant.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make(map[string]int64, len(counts))
	for cid, c := range counts {
		out[strconv.FormatInt(cid, 10)] = c
	}
	return connect.NewResponse(&channelv1.GetChannelUnreadCountsResponse{Unread: out}), nil
}

func (s *Server) MuteChannel(
	ctx context.Context, req *connect.Request[channelv1.MuteChannelRequest],
) (*connect.Response[channelv1.MuteChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.SetMemberMuted(ctx, ch.ID, tenant.UserID, req.Msg.GetMuted()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.MuteChannelResponse{Status: "ok"}), nil
}
