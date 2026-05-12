package channelconnect

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	channeldomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelservice "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100
	defaultSearchLimit   = 20
	maxSearchLimit       = 100
)

func (s *Server) ListChannelMessages(
	ctx context.Context, req *connect.Request[channelv1.ListChannelMessagesRequest],
) (*connect.Response[channelv1.ListChannelMessagesResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	limit := clampLimit(req.Msg.Limit, defaultMessagesLimit, maxMessagesLimit)

	var messages []*channeldomain.Message
	var hasMore bool
	if before := req.Msg.GetBeforeId(); before > 0 {
		messages, hasMore, err = s.channelSvc.GetMessagesByCursor(ctx, ch.ID, before, int(limit))
	} else {
		messages, hasMore, err = s.channelSvc.GetMessages(ctx, ch.ID, nil, nil, int(limit))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*channelv1.ChannelMessage, 0, len(messages))
	for _, m := range messages {
		items = append(items, toProtoMessage(m))
	}
	return connect.NewResponse(&channelv1.ListChannelMessagesResponse{
		Items:   items,
		Total:   int64(len(items)),
		Limit:   limit,
		Offset:  0,
		HasMore: hasMore,
	}), nil
}

func (s *Server) SearchChannelMessages(
	ctx context.Context, req *connect.Request[channelv1.SearchChannelMessagesRequest],
) (*connect.Response[channelv1.SearchChannelMessagesResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	query := req.Msg.GetQuery()
	if query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}
	limit := clampLimit(req.Msg.Limit, defaultSearchLimit, maxSearchLimit)
	messages, err := s.channelSvc.SearchMessages(ctx, ch.ID, query, int(limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*channelv1.ChannelMessage, 0, len(messages))
	for _, m := range messages {
		items = append(items, toProtoMessage(m))
	}
	return connect.NewResponse(&channelv1.SearchChannelMessagesResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  limit,
		Offset: 0,
	}), nil
}

func (s *Server) SendChannelMessage(
	ctx context.Context, req *connect.Request[channelv1.SendChannelMessageRequest],
) (*connect.Response[channelv1.ChannelMessage], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	if ch.IsArchived {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("Cannot send messages to archived channel"))
	}
	content, err := resolveSendContent(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	tenant := middleware.GetTenant(ctx)
	var podKey *string
	if k := req.Msg.GetPodKey(); k != "" {
		podKey = &k
	}
	var replyTo *int64
	if r := req.Msg.GetReplyTo(); r != 0 {
		replyTo = &r
	}
	msg, err := s.channelSvc.SendMessage(ctx, ch.ID, podKey, &tenant.UserID, content, replyTo)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoMessage(msg)), nil
}

func (s *Server) EditChannelMessage(
	ctx context.Context, req *connect.Request[channelv1.EditChannelMessageRequest],
) (*connect.Response[channelv1.ChannelMessage], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	content, err := resolveEditContent(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tenant := middleware.GetTenant(ctx)
	msg, err := s.channelSvc.EditMessage(ctx, ch.ID, req.Msg.GetMessageId(), tenant.UserID, content)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoMessage(msg)), nil
}

func (s *Server) DeleteChannelMessage(
	ctx context.Context, req *connect.Request[channelv1.DeleteChannelMessageRequest],
) (*connect.Response[channelv1.DeleteChannelMessageResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	ch, err := s.requireChannelAccess(ctx, req.Msg.GetChannelId())
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.channelSvc.DeleteMessage(ctx, ch.ID, req.Msg.GetMessageId(), tenant.UserID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&channelv1.DeleteChannelMessageResponse{Status: "deleted"}), nil
}

// resolveSendContent mirrors REST's resolveContent (channel_messages.go:74):
// accept either `source` (markdown) or `content_json` (pre-built AST), but
// not both. attachment_key is permitted on its own (text=empty + attachment).
func resolveSendContent(req *channelv1.SendChannelMessageRequest) (channeldomain.MessageContent, error) {
	return resolveContent(
		req.Source, req.GetMentions(), req.ContentJson, req.GetAttachmentKey(),
	)
}

func resolveEditContent(req *channelv1.EditChannelMessageRequest) (channeldomain.MessageContent, error) {
	return resolveContent(
		req.Source, req.GetMentions(), req.ContentJson, req.GetAttachmentKey(),
	)
}

func resolveContent(
	source *string, mentions map[string]*channelv1.MentionRef,
	contentJSON *string, attachmentKey string,
) (channeldomain.MessageContent, error) {
	hasSource := source != nil && *source != ""
	hasContent := contentJSON != nil && *contentJSON != ""
	if hasSource && hasContent {
		return channeldomain.MessageContent{}, errors.New("provide either source or content_json, not both")
	}
	var resolved channeldomain.MessageContent
	switch {
	case hasSource:
		refs := make(map[string]channelservice.MentionRef, len(mentions))
		for k, v := range mentions {
			if v == nil {
				continue
			}
			refs[k] = channelservice.MentionRef{EntityType: v.GetEntityType(), EntityKey: v.GetEntityKey()}
		}
		parsed, err := channelservice.ParseMarkdown(*source, refs)
		if err != nil {
			return channeldomain.MessageContent{}, err
		}
		resolved = parsed
	case hasContent:
		if err := json.Unmarshal([]byte(*contentJSON), &resolved); err != nil {
			return channeldomain.MessageContent{}, err
		}
	case attachmentKey != "":
		resolved = channeldomain.MessageContent{Kind: "text"}
	default:
		return channeldomain.MessageContent{}, errors.New("source, content_json, or attachment_key is required")
	}
	if attachmentKey != "" {
		resolved.AttachmentKey = attachmentKey
	}
	return resolved, nil
}

func clampLimit(p *int32, defaultVal, max int32) int32 {
	if p == nil || *p <= 0 {
		return defaultVal
	}
	if *p > max {
		return max
	}
	return *p
}
