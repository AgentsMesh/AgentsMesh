package channelconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelservice "github.com/anthropics/agentsmesh/backend/internal/service/channel"
)

// requireChannelAccess fetches a channel, validates org ownership, and
// enforces visibility (private channels require membership). Mirrors REST's
// requireChannelAccess (channel_authz.go:17).
//
// Returns the channel + connect-typed error. Caller must have already passed
// ResolveOrgScope so that middleware.GetTenant(ctx) is populated.
func (s *Server) requireChannelAccess(ctx context.Context, channelID int64) (*channel.Channel, error) {
	if channelID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("channel id is required"))
	}
	ch, err := s.channelSvc.GetChannel(ctx, channelID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	tenant := middleware.GetTenant(ctx)
	if ch.OrganizationID != tenant.OrganizationID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("channel does not belong to this organization"))
	}
	if !ch.IsPublic() {
		ok, err := s.channelSvc.IsMember(ctx, ch.ID, tenant.UserID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !ok {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not a channel member"))
		}
	}
	return ch, nil
}

// mapServiceError mirrors handleChannelServiceError (channel_authz.go:55).
// Translates channel-domain sentinels to Connect codes per conventions §10.
func mapServiceError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, channelservice.ErrChannelNotFound),
		errors.Is(err, channelservice.ErrMessageNotFound):
		return connect.NewError(connect.CodeNotFound, err)

	case errors.Is(err, channelservice.ErrDuplicateName):
		return connect.NewError(connect.CodeAlreadyExists, err)

	case errors.Is(err, channelservice.ErrChannelArchived):
		// REST translates this to 409 for archive-state conflicts and to
		// 400 for archived-channel writes. We pick the stricter 409 here
		// — handler-level wrappers may downgrade for specific RPCs.
		return connect.NewError(connect.CodeFailedPrecondition, err)

	case errors.Is(err, channelservice.ErrNotMember),
		errors.Is(err, channelservice.ErrChannelPrivate),
		errors.Is(err, channelservice.ErrNotCreator),
		errors.Is(err, channelservice.ErrNotMessageSender):
		return connect.NewError(connect.CodePermissionDenied, err)

	case errors.Is(err, channelservice.ErrEmptyContent),
		errors.Is(err, channelservice.ErrInvalidContent):
		return connect.NewError(connect.CodeInvalidArgument, err)

	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
