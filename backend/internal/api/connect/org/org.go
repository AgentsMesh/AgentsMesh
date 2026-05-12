// Package orgconnect hosts Connect-RPC handlers for the organization
// domain. Mirrors backend/internal/api/rest/v1/organizations*.go but exposes
// the data plane via Connect (binary protobuf wire, conventions §2.5).
// REST stays mounted in parallel; the migration runs dual-track until all
// 26 services have flipped.
//
// Mixed-scope service: ListMyOrgs / CreateOrg / CreatePersonalOrg are
// user-scoped (auth interceptor supplies the user, no org_slug payload —
// conventions §3.5 exception #1). Get/Update/Delete/Members go through
// ResolveOrgScope which resolves org_slug + injects TenantContext.
package orgconnect

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	orgservice "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	userservice "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// ServiceName mirrors proto.org.v1.OrgService exactly — Connect derives the
// URL from `<package>.<Service>` (conventions §1, §12).
const ServiceName = "proto.org.v1.OrgService"

const (
	ListMyOrgsProcedure        = "/" + ServiceName + "/ListMyOrgs"
	CreateOrgProcedure         = "/" + ServiceName + "/CreateOrg"
	CreatePersonalOrgProcedure = "/" + ServiceName + "/CreatePersonalOrg"
	GetOrgProcedure            = "/" + ServiceName + "/GetOrg"
	UpdateOrgProcedure         = "/" + ServiceName + "/UpdateOrg"
	DeleteOrgProcedure         = "/" + ServiceName + "/DeleteOrg"
	ListMembersProcedure       = "/" + ServiceName + "/ListMembers"
	InviteMemberProcedure      = "/" + ServiceName + "/InviteMember"
	RemoveMemberProcedure      = "/" + ServiceName + "/RemoveMember"
	UpdateMemberRoleProcedure  = "/" + ServiceName + "/UpdateMemberRole"
)

// Server implements the OrgService contract. orgSvc owns the membership
// queries; userSvc owns username/email lookup paths for CreatePersonalOrg
// and email-based invites.
type Server struct {
	orgSvc  *orgservice.Service
	userSvc *userservice.Service
}

func NewServer(orgSvc *orgservice.Service, userSvc *userservice.Service) *Server {
	return &Server{orgSvc: orgSvc, userSvc: userSvc}
}

// Mount registers all OrgService procedures on mux behind the auth
// interceptor supplied via opts (see cmd/server/connect_init.go).
func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListMyOrgsProcedure, connect.NewUnaryHandler(
		ListMyOrgsProcedure, srv.ListMyOrgs, opts...,
	))
	mux.Handle(CreateOrgProcedure, connect.NewUnaryHandler(
		CreateOrgProcedure, srv.CreateOrg, opts...,
	))
	mux.Handle(CreatePersonalOrgProcedure, connect.NewUnaryHandler(
		CreatePersonalOrgProcedure, srv.CreatePersonalOrg, opts...,
	))
	mux.Handle(GetOrgProcedure, connect.NewUnaryHandler(
		GetOrgProcedure, srv.GetOrg, opts...,
	))
	mux.Handle(UpdateOrgProcedure, connect.NewUnaryHandler(
		UpdateOrgProcedure, srv.UpdateOrg, opts...,
	))
	mux.Handle(DeleteOrgProcedure, connect.NewUnaryHandler(
		DeleteOrgProcedure, srv.DeleteOrg, opts...,
	))
	mux.Handle(ListMembersProcedure, connect.NewUnaryHandler(
		ListMembersProcedure, srv.ListMembers, opts...,
	))
	mux.Handle(InviteMemberProcedure, connect.NewUnaryHandler(
		InviteMemberProcedure, srv.InviteMember, opts...,
	))
	mux.Handle(RemoveMemberProcedure, connect.NewUnaryHandler(
		RemoveMemberProcedure, srv.RemoveMember, opts...,
	))
	mux.Handle(UpdateMemberRoleProcedure, connect.NewUnaryHandler(
		UpdateMemberRoleProcedure, srv.UpdateMemberRole, opts...,
	))
}

// requireAdmin checks that the tenant's role allows admin actions (owner or
// admin). Mirrors REST's IsAdmin path on organizations_members.go.
func requireAdmin(ctx context.Context) error {
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

// requireOwner checks that the tenant's role is owner only. Mirrors REST's
// IsOwner path used on DeleteOrganization.
func requireOwner(ctx context.Context) error {
	tenant := middleware.GetTenant(ctx)
	if tenant == nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("missing tenant context"))
	}
	if tenant.UserRole != "owner" {
		return connect.NewError(
			connect.CodePermissionDenied,
			errors.New("organization owner role required"),
		)
	}
	return nil
}

// userIDFromCtx pulls the authenticated user ID from the TenantContext the
// auth interceptor populated. Used by user-scoped RPCs that don't go through
// ResolveOrgScope. Returns CodeUnauthenticated when missing.
func userIDFromCtx(ctx context.Context) (int64, error) {
	tenant := middleware.GetTenant(ctx)
	if tenant == nil || tenant.UserID == 0 {
		return 0, connect.NewError(connect.CodeUnauthenticated,
			errors.New("authentication required"))
	}
	return tenant.UserID, nil
}

// mapServiceError translates org-service sentinels to Connect codes per
// conventions §10.
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, orgservice.ErrOrganizationNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, orgservice.ErrSlugAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, orgservice.ErrNotOrganizationAdmin):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, orgservice.ErrCannotRemoveOwner):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
