package apikeyconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
)

// requireOrgAdmin gates write/read access at the connect handler layer
// — apikeys are sensitive secrets, REST uses the same admin gate
// (apikey_routes.go).
func requireOrgAdmin(ctx context.Context) error {
	tenant := middleware.GetTenant(ctx)
	if tenant == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing tenant context"))
	}
	if tenant.UserRole != "admin" && tenant.UserRole != "owner" {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("organization admin role required"),
		)
	}
	return nil
}

// mapServiceError mirrors handleAPIKeyServiceError (apikeys.go:163).
// Translates apikey-domain sentinels to Connect codes per conventions §10.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, apikeyservice.ErrAPIKeyNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, apikeyservice.ErrDuplicateKeyName):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, apikeyservice.ErrNameEmpty),
		errors.Is(err, apikeyservice.ErrNameTooLong),
		errors.Is(err, apikeyservice.ErrScopesRequired),
		errors.Is(err, apikeyservice.ErrInvalidScope),
		errors.Is(err, apikeyservice.ErrInvalidExpiresIn):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
