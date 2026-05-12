package repositoryconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	repositoryv1 "github.com/anthropics/agentsmesh/proto/gen/go/repository/v1"
)

// UpdateRepository mirrors REST handler `UpdateRepository`
// (repositories_crud.go:140). admin + write-policy gate; partial updates
// honour each optional proto3 field's presence.
func (s *Server) UpdateRepository(
	ctx context.Context, req *connect.Request[repositoryv1.UpdateRepositoryRequest],
) (*connect.Response[repositoryv1.Repository], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin role required"))
	}

	repo, err := s.repoSvc.GetByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapServiceError(err)
	}
	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
	}

	updates := buildUpdateMap(req.Msg)
	repo, err = s.repoSvc.Update(ctx, req.Msg.GetId(), updates)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoRepository(repo)), nil
}

// DeleteRepository mirrors REST handler `DeleteRepository`
// (repositories_crud.go:205). Cleans up grants on success.
func (s *Server) DeleteRepository(
	ctx context.Context, req *connect.Request[repositoryv1.DeleteRepositoryRequest],
) (*connect.Response[repositoryv1.DeleteRepositoryResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin role required"))
	}

	repo, err := s.repoSvc.GetByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapServiceError(err)
	}
	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
	}

	if err := s.repoSvc.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, mapServiceError(err)
	}
	if s.grantSvc != nil {
		_ = s.grantSvc.CleanupByResource(ctx, grant.TypeRepository, grant.IntResourceID(req.Msg.GetId()))
	}
	return connect.NewResponse(&repositoryv1.DeleteRepositoryResponse{}), nil
}

// buildUpdateMap mirrors REST's partial-update map (repositories_crud.go:174).
// proto3 optional semantics: only fields whose `Has*()` accessor returns
// true land in the update map. Empty strings via the legacy non-optional
// path stay out (matches REST behavior).
func buildUpdateMap(req *repositoryv1.UpdateRepositoryRequest) map[string]interface{} {
	updates := make(map[string]interface{})
	if v := req.GetName(); v != "" {
		updates["name"] = v
	}
	if v := req.GetDefaultBranch(); v != "" {
		updates["default_branch"] = v
	}
	if v := req.GetTicketPrefix(); v != "" {
		updates["ticket_prefix"] = v
	}
	if req.IsActive != nil {
		updates["is_active"] = req.GetIsActive()
	}
	if req.HttpCloneUrl != nil {
		updates["http_clone_url"] = req.GetHttpCloneUrl()
	}
	if req.SshCloneUrl != nil {
		updates["ssh_clone_url"] = req.GetSshCloneUrl()
	}
	return updates
}
