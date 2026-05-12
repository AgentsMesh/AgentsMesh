package repositoryconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	repositoryservice "github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	repositoryv1 "github.com/anthropics/agentsmesh/proto/gen/go/repository/v1"
)

// ListRepositories mirrors REST handler `ListRepositories` (repositories_crud.go:19).
// Applies the same visibility filter (`policy.RepositoryPolicy.ListFilter`).
func (s *Server) ListRepositories(
	ctx context.Context, req *connect.Request[repositoryv1.ListRepositoriesRequest],
) (*connect.Response[repositoryv1.ListRepositoriesResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	filter := policy.RepositoryPolicy.ListFilter(sub)

	repos, err := s.repoSvc.ListByOrganizationForUser(
		ctx, tenant.OrganizationID, filter.VisibilityUserID,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*repositoryv1.Repository, 0, len(repos))
	for _, r := range repos {
		items = append(items, toProtoRepository(r))
	}
	return connect.NewResponse(&repositoryv1.ListRepositoriesResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  req.Msg.GetLimit(),
		Offset: req.Msg.GetOffset(),
	}), nil
}

// GetRepository mirrors REST handler `GetRepository` (repositories_crud.go:113).
func (s *Server) GetRepository(
	ctx context.Context, req *connect.Request[repositoryv1.GetRepositoryRequest],
) (*connect.Response[repositoryv1.Repository], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	repo, err := s.repoSvc.GetByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapServiceError(err)
	}

	tenant := middleware.GetTenant(ctx)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.RepositoryPolicy.AllowRead(sub, s.resourceWithGrants(
		ctx, repo.ID, repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
	}

	return connect.NewResponse(toProtoRepository(repo)), nil
}

// CreateRepository mirrors REST handler `CreateRepository`
// (repositories_crud.go:35). Performs the billing-quota check + admin gate
// before delegating to the service layer.
func (s *Server) CreateRepository(
	ctx context.Context, req *connect.Request[repositoryv1.CreateRepositoryRequest],
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

	if err := s.checkRepositoryQuota(ctx, tenant.OrganizationID, req.Msg); err != nil {
		return nil, err
	}

	repo, err := s.repoSvc.Create(ctx, buildCreateRequest(tenant, req.Msg))
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoRepository(repo)), nil
}

// checkRepositoryQuota replicates the REST handler's quota guard
// (repositories_crud.go:52). Only enforced on new repositories; re-imports
// of existing rows skip the count to match historical REST semantics.
func (s *Server) checkRepositoryQuota(
	ctx context.Context, orgID int64, req *repositoryv1.CreateRepositoryRequest,
) error {
	if s.billingSvc == nil {
		return nil
	}
	_, existsErr := s.repoSvc.GetBySlug(
		ctx, orgID,
		req.GetProviderType(), req.GetProviderBaseUrl(), req.GetSlug(),
	)
	if !errors.Is(existsErr, repositoryservice.ErrRepositoryNotFound) {
		return nil
	}
	if err := s.billingSvc.CheckQuota(ctx, orgID, "repositories", 1); err != nil {
		return mapBillingError(err)
	}
	return nil
}

// buildCreateRequest applies defaults that the REST handler used to inline.
func buildCreateRequest(
	tenant *middleware.TenantContext, req *repositoryv1.CreateRepositoryRequest,
) *repositoryservice.CreateRequest {
	defaultBranch := req.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	visibility := req.GetVisibility()
	if visibility == "" {
		visibility = "organization"
	}
	var ticketPrefix *string
	if tp := req.GetTicketPrefix(); tp != "" {
		ticketPrefix = &tp
	}
	userID := tenant.UserID
	return &repositoryservice.CreateRequest{
		OrganizationID:   tenant.OrganizationID,
		ProviderType:     req.GetProviderType(),
		ProviderBaseURL:  req.GetProviderBaseUrl(),
		HttpCloneURL:     req.GetHttpCloneUrl(),
		SshCloneURL:      req.GetSshCloneUrl(),
		ExternalID:       req.GetExternalId(),
		Name:             req.GetName(),
		Slug:             req.GetSlug(),
		DefaultBranch:    defaultBranch,
		TicketPrefix:     ticketPrefix,
		Visibility:       visibility,
		ImportedByUserID: &userID,
	}
}