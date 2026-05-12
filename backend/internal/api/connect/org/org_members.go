package orgconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	orgv1 "github.com/anthropics/agentsmesh/proto/gen/go/org/v1"
)

// ListMembers returns the org's members with user details. Membership check
// runs through ResolveOrgScope. Mirrors REST GET /api/v1/orgs/:slug/members
// (organizations_members.go:16-40).
func (s *Server) ListMembers(
	ctx context.Context, req *connect.Request[orgv1.ListMembersRequest],
) (*connect.Response[orgv1.ListMembersResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)
	members, err := s.orgSvc.ListMembers(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*orgv1.OrganizationMember, 0, len(members))
	for _, m := range members {
		items = append(items, toProtoMember(m))
	}
	return connect.NewResponse(&orgv1.ListMembersResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  req.Msg.GetLimit(),
		Offset: req.Msg.GetOffset(),
	}), nil
}

// InviteMember adds a member to the org. Supports both email-based invite
// (resolves to user_id via userSvc.GetByEmail) and direct user_id addition.
// Admin role required. Mirrors REST POST /api/v1/orgs/:slug/members
// (organizations_members.go:45-97).
func (s *Server) InviteMember(
	ctx context.Context, req *connect.Request[orgv1.InviteMemberRequest],
) (*connect.Response[orgv1.InviteMemberResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if role := req.Msg.GetRole(); role != "admin" && role != "member" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("role must be admin or member"))
	}

	targetUserID := req.Msg.GetUserId()
	if email := req.Msg.GetEmail(); email != "" {
		u, err := s.userSvc.GetByEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound,
				errors.New("user not found with this email"))
		}
		targetUserID = u.ID
	}
	if targetUserID == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("either email or user_id is required"))
	}

	tenant := middleware.GetTenant(ctx)

	isMember, _ := s.orgSvc.IsMember(ctx, tenant.OrganizationID, targetUserID)
	if isMember {
		return nil, connect.NewError(connect.CodeAlreadyExists,
			errors.New("user is already a member of this organization"))
	}
	if err := s.orgSvc.AddMember(ctx, tenant.OrganizationID, targetUserID, req.Msg.GetRole()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&orgv1.InviteMemberResponse{Message: "Member added"}), nil
}

// RemoveMember removes a member from the org. Cannot remove the owner.
// Admin role required. Mirrors REST DELETE /api/v1/orgs/:slug/members/:user_id
// (organizations_members.go:101-133).
func (s *Server) RemoveMember(
	ctx context.Context, req *connect.Request[orgv1.RemoveMemberRequest],
) (*connect.Response[orgv1.RemoveMemberResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if req.Msg.GetUserId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("user_id is required"))
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.orgSvc.RemoveMember(ctx, tenant.OrganizationID, req.Msg.GetUserId()); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&orgv1.RemoveMemberResponse{Message: "Member removed"}), nil
}

// UpdateMemberRole changes a member's role. Admin role required. Mirrors
// REST PUT /api/v1/orgs/:slug/members/:user_id
// (organizations_members.go:137-171).
func (s *Server) UpdateMemberRole(
	ctx context.Context, req *connect.Request[orgv1.UpdateMemberRoleRequest],
) (*connect.Response[orgv1.UpdateMemberRoleResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if role := req.Msg.GetRole(); role != "admin" && role != "member" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("role must be admin or member"))
	}
	if req.Msg.GetUserId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("user_id is required"))
	}
	tenant := middleware.GetTenant(ctx)
	if err := s.orgSvc.UpdateMemberRole(
		ctx, tenant.OrganizationID, req.Msg.GetUserId(), req.Msg.GetRole(),
	); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&orgv1.UpdateMemberRoleResponse{Message: "Role updated"}), nil
}
