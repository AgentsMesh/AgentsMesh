package channelconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

func (s *Server) GetChannelDocument(
	ctx context.Context, req *connect.Request[channelv1.GetChannelDocumentRequest],
) (*connect.Response[channelv1.GetChannelDocumentResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	doc := ""
	if ch.Document != nil {
		doc = *ch.Document
	}
	return connect.NewResponse(&channelv1.GetChannelDocumentResponse{Document: doc}), nil
}

func (s *Server) UpdateChannelDocument(
	ctx context.Context, req *connect.Request[channelv1.UpdateChannelDocumentRequest],
) (*connect.Response[channelv1.UpdateChannelDocumentResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	doc := req.Msg.GetDocument()
	if _, err := s.channelSvc.UpdateChannel(ctx, ch.ID, nil, nil, &doc); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.UpdateChannelDocumentResponse{Document: doc}), nil
}
