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
	summaries, err := s.channelSvc.GetChannelSummaries(ctx, tenant.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	unread := make(map[string]int64, len(summaries))
	mentions := make(map[string]int64)
	lastRead := make(map[string]int64)
	manuallyUnread := make(map[string]bool)
	for cid, sum := range summaries {
		key := strconv.FormatInt(cid, 10)
		if sum.Unread > 0 {
			unread[key] = sum.Unread
			lastRead[key] = sum.LastRead
		}
		if sum.Mention > 0 {
			mentions[key] = sum.Mention
		}
		if sum.ManuallyUnread {
			manuallyUnread[key] = true
		}
	}
	return connect.NewResponse(&channelv1.GetChannelUnreadCountsResponse{
		Unread:         unread,
		Mentions:       mentions,
		LastRead:       lastRead,
		ManuallyUnread: manuallyUnread,
	}), nil
}

func (s *Server) MarkChannelUnread(
	ctx context.Context, req *connect.Request[channelv1.MarkChannelUnreadRequest],
) (*connect.Response[channelv1.MarkChannelUnreadResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.SetManuallyUnread(ctx, ch.ID, tenant.UserID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.MarkChannelUnreadResponse{Status: "ok"}), nil
}

func (s *Server) GetMessageReadBy(
	ctx context.Context, req *connect.Request[channelv1.GetMessageReadByRequest],
) (*connect.Response[channelv1.GetMessageReadByResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	ids, err := s.channelSvc.GetMessageReadBy(ctx, ch.ID, req.Msg.GetMessageId(), tenant.UserID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	// "Read by" lists who *else* read it — drop the requester (message author).
	readBy := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != tenant.UserID {
			readBy = append(readBy, id)
		}
	}
	return connect.NewResponse(&channelv1.GetMessageReadByResponse{UserIds: readBy}), nil
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

func (s *Server) PinChannel(
	ctx context.Context, req *connect.Request[channelv1.PinChannelRequest],
) (*connect.Response[channelv1.PinChannelResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.SetMemberPinned(ctx, ch.ID, tenant.UserID, req.Msg.GetPinned()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.PinChannelResponse{Status: "ok"}), nil
}
