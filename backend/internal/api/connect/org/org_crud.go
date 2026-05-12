package orgconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	orgv1 "github.com/anthropics/agentsmesh/proto/gen/go/org/v1"
)

// GetOrg returns an organization by slug. Membership is verified by the
// ResolveOrgScope helper. Mirrors REST GET /api/v1/orgs/:slug
// (organizations_crud.go:67-89).
func (s *Server) GetOrg(
	ctx context.Context, req *connect.Request[orgv1.GetOrgRequest],
) (*connect.Response[orgv1.Organization], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	org, err := s.orgSvc.GetOrgBySlug(ctx, tenant.OrganizationSlug)
	if err != nil {
		return nil, mapServiceError(err)
	}
	// Preserve the requesting user's role from TenantContext so the
	// response carries it (the REST handler doesn't populate Role here,
	// but ListMyOrgs does — keeping the field consistent across endpoints).
	out := toProtoOrganization(org)
	if tenant.UserRole != "" {
		role := tenant.UserRole
		out.Role = &role
	}
	return connect.NewResponse(out), nil
}

// UpdateOrg updates an organization. Admin role required. Mirrors REST PUT
// /api/v1/orgs/:slug (organizations_crud.go:93-135).
func (s *Server) UpdateOrg(
	ctx context.Context, req *connect.Request[orgv1.UpdateOrgRequest],
) (*connect.Response[orgv1.Organization], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	updates := make(map[string]interface{})
	if name := req.Msg.GetName(); name != "" {
		updates["name"] = name
	}
	if logo := req.Msg.GetLogoUrl(); logo != "" {
		updates["logo_url"] = logo
	}

	org, err := s.orgSvc.Update(ctx, tenant.OrganizationID, updates)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoOrganization(org)), nil
}

// DeleteOrg deletes an organization. Owner role required. Mirrors REST
// DELETE /api/v1/orgs/:slug (organizations_crud.go:139-166).
func (s *Server) DeleteOrg(
	ctx context.Context, req *connect.Request[orgv1.DeleteOrgRequest],
) (*connect.Response[orgv1.DeleteOrgResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOwner(ctx); err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.orgSvc.Delete(ctx, tenant.OrganizationID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&orgv1.DeleteOrgResponse{
		Message: "Organization deleted",
	}), nil
}
