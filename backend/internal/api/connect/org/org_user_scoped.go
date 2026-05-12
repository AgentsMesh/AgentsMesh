package orgconnect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	orgservice "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
	orgv1 "github.com/anthropics/agentsmesh/proto/gen/go/org/v1"
)

// ListMyOrgs returns the caller's organizations. User-scoped — auth
// interceptor populates user_id; no org_slug payload (conventions §3.5
// exception #1). Mirrors REST GET /api/v1/organizations
// (organizations_crud.go:16-26).
func (s *Server) ListMyOrgs(
	ctx context.Context, _ *connect.Request[orgv1.ListMyOrgsRequest],
) (*connect.Response[orgv1.ListMyOrgsResponse], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	orgs, err := s.orgSvc.ListByUser(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*orgv1.Organization, 0, len(orgs))
	for _, o := range orgs {
		items = append(items, toProtoOrganization(o))
	}
	return connect.NewResponse(&orgv1.ListMyOrgsResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

// CreateOrg creates a new organization owned by the caller. User-scoped —
// caller-supplied slug is validated against slugkit rules. Mirrors REST
// POST /api/v1/organizations (organizations_crud.go:30-63).
func (s *Server) CreateOrg(
	ctx context.Context, req *connect.Request[orgv1.CreateOrgRequest],
) (*connect.Response[orgv1.Organization], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("name is required"))
	}
	if req.Msg.GetSlug() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("slug is required"))
	}
	if err := slugkit.Validate(req.Msg.GetSlug()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	org, err := s.orgSvc.Create(ctx, userID, &orgservice.CreateRequest{
		Name:    req.Msg.GetName(),
		Slug:    req.Msg.GetSlug(),
		LogoURL: req.Msg.GetLogoUrl(),
	})
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoOrganization(org)), nil
}

// CreatePersonalOrg creates the caller's personal workspace. Slug is derived
// server-side from username (collision-resistant). Mirrors REST POST
// /api/v1/orgs/personal (organizations_personal.go:20-58).
func (s *Server) CreatePersonalOrg(
	ctx context.Context, _ *connect.Request[orgv1.CreatePersonalOrgRequest],
) (*connect.Response[orgv1.Organization], error) {
	userID, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.userSvc.GetByID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "create personal: user lookup failed",
			"user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to load user"))
	}
	displayName := ""
	if u.Name != nil {
		displayName = *u.Name
	}
	org, err := s.orgSvc.CreatePersonal(ctx, userID, u.Username, displayName)
	if err != nil {
		switch {
		case errors.Is(err, slugkit.ErrCollisionExhausted):
			slog.ErrorContext(ctx, "create personal: slug collision exhausted",
				"user_id", userID, "username", u.Username, "error", err)
			return nil, connect.NewError(connect.CodeResourceExhausted,
				errors.New("could not allocate a unique workspace slug after retries"))
		case errors.Is(err, orgservice.ErrSlugAlreadyExists):
			slog.ErrorContext(ctx, "create personal: race lost on slug insert",
				"user_id", userID, "username", u.Username, "error", err)
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		default:
			slog.ErrorContext(ctx, "create personal: unexpected failure",
				"user_id", userID, "username", u.Username, "error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(toProtoOrganization(org)), nil
}
