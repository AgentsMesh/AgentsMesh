package channelconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

func (s *Server) ListChannelPods(
	ctx context.Context, req *connect.Request[channelv1.ListChannelPodsRequest],
) (*connect.Response[channelv1.ListChannelPodsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	pods, err := s.channelSvc.GetChannelPods(ctx, ch.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*channelv1.ChannelPod, 0, len(pods))
	for _, p := range pods {
		items = append(items, toProtoChannelPod(p))
	}
	return connect.NewResponse(&channelv1.ListChannelPodsResponse{
		Items: items,
		Total: int64(len(items)),
		Limit: 0, Offset: 0,
	}), nil
}

func (s *Server) JoinChannelPod(
	ctx context.Context, req *connect.Request[channelv1.JoinChannelPodRequest],
) (*connect.Response[channelv1.JoinChannelPodResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	podKey := req.Msg.GetPodKey()
	if podKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pod_key is required"))
	}
	if err := s.channelSvc.JoinChannel(ctx, ch.ID, podKey); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.JoinChannelPodResponse{Message: "Pod joined channel"}), nil
}

func (s *Server) LeaveChannelPod(
	ctx context.Context, req *connect.Request[channelv1.LeaveChannelPodRequest],
) (*connect.Response[channelv1.LeaveChannelPodResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.channelSvc.LeaveChannel(ctx, ch.ID, req.Msg.GetPodKey()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.LeaveChannelPodResponse{Message: "Pod left channel"}), nil
}
