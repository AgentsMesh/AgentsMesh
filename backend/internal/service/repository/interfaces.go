package repository

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

// RepositoryServiceInterface defines the interface for repository operations
// This interface enables mocking in tests
type RepositoryServiceInterface interface {
	// GetByID returns a repository by ID
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
	// GetByIDForUser returns a repository by ID with visibility checks
	GetByIDForUser(ctx context.Context, id int64, userID int64) (*gitprovider.Repository, error)
	// Create creates a new repository
	Create(ctx context.Context, req *CreateRequest) (*gitprovider.Repository, error)
	// CreateWithWebhook creates a repository and registers a webhook
	CreateWithWebhook(ctx context.Context, req *CreateRequest, orgSlug string) (*gitprovider.Repository, *WebhookResult, error)
	// Update updates a repository
	Update(ctx context.Context, id int64, updates map[string]interface{}) (*gitprovider.Repository, error)
	// Delete soft deletes a repository
	Delete(ctx context.Context, id int64) error
	// ListByOrganization returns repositories for an organization
	ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error)
	// ListByOrganizationForUser returns repositories visible to a specific user
	ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error)
	// GetWebhookService returns the webhook service
	GetWebhookService() WebhookServiceInterface
	// ListBranches lists branches for a repository
	ListBranches(ctx context.Context, repoID int64, accessToken string) ([]string, error)
	// SyncFromProvider syncs repository info from git provider
	SyncFromProvider(ctx context.Context, repoID int64, accessToken string) (*gitprovider.Repository, error)
	// GetBySlug returns a repository by org, provider, and slug
	GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error)
	// ListMergeRequests lists merge requests for a repository
	ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]*MergeRequestInfo, error)
}

// WebhookServiceInterface defines the interface for webhook operations
// This interface enables mocking in tests
type WebhookServiceInterface interface {
	// RegisterWebhookForRepository registers a webhook for a repository
	RegisterWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, orgSlug string, userID int64) (*WebhookResult, error)
	// DeleteWebhookForRepository deletes a webhook from a repository
	DeleteWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, userID int64) error
	// GetWebhookStatus returns the webhook status for a repository
	GetWebhookStatus(ctx context.Context, repo *gitprovider.Repository) *gitprovider.WebhookStatus
	// GetWebhookSecret returns the webhook secret for manual configuration
	GetWebhookSecret(ctx context.Context, repo *gitprovider.Repository) (string, error)
	// MarkWebhookAsConfigured marks a webhook as manually configured
	MarkWebhookAsConfigured(ctx context.Context, repo *gitprovider.Repository) error
	// VerifyWebhookSecret verifies the webhook secret for a repository
	VerifyWebhookSecret(ctx context.Context, repoID int64, secret string) (bool, error)
	// GetRepositoryByIDWithWebhook returns a repository by ID with webhook config
	GetRepositoryByIDWithWebhook(ctx context.Context, repoID int64) (*gitprovider.Repository, error)
}

// Ensure Service implements RepositoryServiceInterface
var _ RepositoryServiceInterface = (*Service)(nil)

// Ensure WebhookService implements WebhookServiceInterface
var _ WebhookServiceInterface = (*WebhookService)(nil)
