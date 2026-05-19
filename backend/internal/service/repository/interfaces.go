package repository

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

type RepositoryServiceInterface interface {
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
	GetByIDForUser(ctx context.Context, id int64, userID int64) (*gitprovider.Repository, error)
	Create(ctx context.Context, req *CreateRequest) (*gitprovider.Repository, error)
	CreateWithWebhook(ctx context.Context, req *CreateRequest, orgSlug string) (*gitprovider.Repository, *WebhookResult, error)
	Update(ctx context.Context, id int64, updates map[string]interface{}) (*gitprovider.Repository, error)
	Delete(ctx context.Context, id int64) error
	ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error)
	ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error)
	GetWebhookService() WebhookServiceInterface
	ListBranches(ctx context.Context, repoID int64, accessToken string) ([]string, error)
	SyncFromProvider(ctx context.Context, repoID int64, accessToken string) (*gitprovider.Repository, error)
	GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error)
	ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]*MergeRequestInfo, error)
}

type WebhookServiceInterface interface {
	RegisterWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, orgSlug string, userID int64) (*WebhookResult, error)
	DeleteWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, userID int64) error
	GetWebhookStatus(ctx context.Context, repo *gitprovider.Repository) *gitprovider.WebhookStatus
	GetWebhookSecret(ctx context.Context, repo *gitprovider.Repository) (string, error)
	MarkWebhookAsConfigured(ctx context.Context, repo *gitprovider.Repository) error
	VerifyWebhookSecret(ctx context.Context, repoID int64, secret string) (bool, error)
	GetRepositoryByIDWithWebhook(ctx context.Context, repoID int64) (*gitprovider.Repository, error)
}

var _ RepositoryServiceInterface = (*Service)(nil)

var _ WebhookServiceInterface = (*WebhookService)(nil)
