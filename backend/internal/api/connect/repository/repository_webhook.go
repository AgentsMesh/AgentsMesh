package repositoryconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	repositoryservice "github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	repositoryv1 "github.com/anthropics/agentsmesh/proto/gen/go/repository/v1"
)

// RegisterRepositoryWebhook mirrors REST handler `RegisterRepositoryWebhook`
// (repositories_webhook.go:17).
func (s *Server) RegisterRepositoryWebhook(
	ctx context.Context, req *connect.Request[repositoryv1.RegisterRepositoryWebhookRequest],
) (*connect.Response[repositoryv1.RegisterRepositoryWebhookResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireRepoWrite(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	webhookSvc := s.repoSvc.GetWebhookService()
	if webhookSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("webhook service not available"))
	}
	result, err := webhookSvc.RegisterWebhookForRepository(
		ctx, repo, tenant.OrganizationSlug, tenant.UserID,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&repositoryv1.RegisterRepositoryWebhookResponse{
		Result: toProtoWebhookResult(result),
	}), nil
}

// DeleteRepositoryWebhook mirrors REST handler `DeleteRepositoryWebhook`
// (repositories_webhook.go:63).
func (s *Server) DeleteRepositoryWebhook(
	ctx context.Context, req *connect.Request[repositoryv1.DeleteRepositoryWebhookRequest],
) (*connect.Response[repositoryv1.DeleteRepositoryWebhookResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireRepoWrite(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	webhookSvc := s.repoSvc.GetWebhookService()
	if webhookSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("webhook service not available"))
	}
	if err := webhookSvc.DeleteWebhookForRepository(ctx, repo, tenant.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&repositoryv1.DeleteRepositoryWebhookResponse{}), nil
}

// GetRepositoryWebhookStatus mirrors REST handler
// `GetRepositoryWebhookStatus` (repositories_webhook.go:108). Read-policy
// gate.
func (s *Server) GetRepositoryWebhookStatus(
	ctx context.Context, req *connect.Request[repositoryv1.GetRepositoryWebhookStatusRequest],
) (*connect.Response[repositoryv1.WebhookStatus], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := s.requireRepoRead(ctx, req.Msg.GetId()); err != nil {
		return nil, err
	}

	repo, err := s.repoSvc.GetByID(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapServiceError(err)
	}
	webhookSvc := s.repoSvc.GetWebhookService()
	if webhookSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("webhook service not available"))
	}
	status := webhookSvc.GetWebhookStatus(ctx, repo)
	return connect.NewResponse(toProtoWebhookStatus(status)), nil
}

// GetRepositoryWebhookSecret mirrors REST handler
// `GetRepositoryWebhookSecret` (repositories_webhook.go:143). Admin-write
// gate; the secret is the most sensitive field on the surface.
func (s *Server) GetRepositoryWebhookSecret(
	ctx context.Context, req *connect.Request[repositoryv1.GetRepositoryWebhookSecretRequest],
) (*connect.Response[repositoryv1.WebhookSecret], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireRepoWrite(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	webhookSvc := s.repoSvc.GetWebhookService()
	if webhookSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("webhook service not available"))
	}
	secret, err := webhookSvc.GetWebhookSecret(ctx, repo)
	if err != nil {
		if errors.Is(err, repositoryservice.ErrWebhookNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &repositoryv1.WebhookSecret{
		WebhookSecret: secret,
	}
	if repo.WebhookConfig != nil {
		out.WebhookUrl = repo.WebhookConfig.URL
		out.Events = repo.WebhookConfig.Events
	}
	return connect.NewResponse(out), nil
}

// MarkRepositoryWebhookConfigured mirrors REST handler
// `MarkRepositoryWebhookConfigured` (repositories_webhook.go:196).
func (s *Server) MarkRepositoryWebhookConfigured(
	ctx context.Context, req *connect.Request[repositoryv1.MarkRepositoryWebhookConfiguredRequest],
) (*connect.Response[repositoryv1.MarkRepositoryWebhookConfiguredResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireRepoWrite(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}

	webhookSvc := s.repoSvc.GetWebhookService()
	if webhookSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("webhook service not available"))
	}
	if err := webhookSvc.MarkWebhookAsConfigured(ctx, repo); err != nil {
		if errors.Is(err, repositoryservice.ErrWebhookNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&repositoryv1.MarkRepositoryWebhookConfiguredResponse{}), nil
}

// requireRepoWrite mirrors the admin + write-policy gate used by webhook
// endpoints. Returns the fetched repository on success so the caller can
// reuse it without re-querying.
func (s *Server) requireRepoWrite(
	ctx context.Context, repoID int64,
) (*gitprovider.Repository, error) {
	tenant := middleware.GetTenant(ctx)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin role required"))
	}
	repo, err := s.repoSvc.GetByID(ctx, repoID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("access denied"))
	}
	return repo, nil
}
