package supportticketconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	supportticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
)

// userIDFromCtx extracts the authenticated user ID for this user-scoped
// service. Mirrors invitation_errors.go's helper — kept local to avoid a
// new shared package while only two services need it.
func userIDFromCtx(ctx context.Context) (int64, error) {
	tenant := middleware.GetTenant(ctx)
	if tenant == nil || tenant.UserID == 0 {
		return 0, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}
	return tenant.UserID, nil
}

// mapSupportTicketError translates support-ticket sentinels to Connect
// codes per conventions §10. Mirrors REST error switches in
// support_tickets.go.
func mapSupportTicketError(err error) error {
	switch {
	case errors.Is(err, supportticketsvc.ErrTicketNotFound),
		errors.Is(err, supportticketsvc.ErrAttachmentNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, supportticketsvc.ErrAccessDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, supportticketsvc.ErrInvalidCategory),
		errors.Is(err, supportticketsvc.ErrInvalidPriority),
		errors.Is(err, supportticketsvc.ErrInvalidStatus),
		errors.Is(err, supportticketsvc.ErrInvalidTransition):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, supportticketsvc.ErrFileTooLarge):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, supportticketsvc.ErrStorageError):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
