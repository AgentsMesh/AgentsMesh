package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

var (
	ErrRepositoryNotFound    = errors.New("repository not found")
	ErrRepositoryExists      = errors.New("repository already exists")
	ErrNoPermission          = errors.New("no permission to access this repository")
	ErrRepositoryHasLoopRefs = errors.New("cannot delete: repository is referenced by one or more loops")
)

// Service handles repository operations
type Service struct {
	repo           gitprovider.RepositoryRepo
	webhookService *WebhookService
}

// NewService creates a new repository service
func NewService(repo gitprovider.RepositoryRepo) *Service {
	return &Service{
		repo: repo,
	}
}

// SetWebhookService sets the webhook service for automatic webhook registration
// This is set separately to avoid circular dependencies during initialization
func (s *Service) SetWebhookService(ws *WebhookService) {
	s.webhookService = ws
}

// GetWebhookService returns the webhook service
func (s *Service) GetWebhookService() WebhookServiceInterface {
	if s.webhookService == nil {
		return nil
	}
	return s.webhookService
}

// CreateRequest represents repository creation request
// Self-contained: no git_provider_id, includes all necessary info
type CreateRequest struct {
	OrganizationID   int64
	ProviderType     string // github, gitlab, gitee, generic
	ProviderBaseURL  string // https://github.com, https://gitlab.company.com
	CloneURL         string // Full clone URL
	HttpCloneURL     string // HTTPS clone URL
	SshCloneURL      string // SSH clone URL
	ExternalID       string
	Name             string
	Slug             string
	DefaultBranch    string
	TicketPrefix     *string
	Visibility       string // "organization" or "private"
	ImportedByUserID *int64 // User who imported this repo
}

// GetByID returns a repository by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error) {
	repo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepositoryNotFound
	}
	return repo, nil
}

// GetByIDForUser returns a repository by ID, checking visibility permissions
func (s *Service) GetByIDForUser(ctx context.Context, id int64, userID int64) (*gitprovider.Repository, error) {
	repo, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check visibility permissions
	if repo.Visibility == "private" {
		if repo.ImportedByUserID == nil || *repo.ImportedByUserID != userID {
			return nil, ErrNoPermission
		}
	}

	return repo, nil
}

// Update updates a repository
func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*gitprovider.Repository, error) {
	if err := s.repo.Update(ctx, id, updates); err != nil {
		slog.Error("failed to update repository", "repo_id", id, "error", err)
		return nil, err
	}
	slog.Info("repository updated", "repo_id", id)
	return s.GetByID(ctx, id)
}

// Delete soft deletes a repository.
// Blocks deletion if any loops reference this repository (application-level RESTRICT).
func (s *Service) Delete(ctx context.Context, id int64) error {
	loopCount, err := s.repo.CountLoopRefs(ctx, id)
	if err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrRepositoryHasLoopRefs
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		slog.Error("failed to soft-delete repository", "repo_id", id, "error", err)
		return err
	}
	slog.Info("repository soft-deleted", "repo_id", id)
	return nil
}

// HardDelete permanently deletes a repository.
// Blocks deletion if any loops reference this repository (application-level RESTRICT).
func (s *Service) HardDelete(ctx context.Context, id int64) error {
	loopCount, err := s.repo.CountLoopRefs(ctx, id)
	if err != nil {
		return err
	}
	if loopCount > 0 {
		return ErrRepositoryHasLoopRefs
	}
	if err := s.repo.HardDelete(ctx, id); err != nil {
		slog.Error("failed to hard-delete repository", "repo_id", id, "error", err)
		return err
	}
	slog.Info("repository hard-deleted", "repo_id", id)
	return nil
}

// ListByOrganization returns repositories for an organization
func (s *Service) ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error) {
	return s.repo.ListByOrganization(ctx, orgID)
}

// ListByOrganizationForUser returns repositories visible to a specific user
func (s *Service) ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error) {
	return s.repo.ListByOrganizationForUser(ctx, orgID, userID)
}

// GetByExternalID returns a repository by provider type, base URL, and external ID
func (s *Service) GetByExternalID(ctx context.Context, providerType, providerBaseURL, externalID string) (*gitprovider.Repository, error) {
	repo, err := s.repo.GetByExternalID(ctx, providerType, providerBaseURL, externalID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepositoryNotFound
	}
	return repo, nil
}

// GetBySlug returns a repository by organization, provider, and slug
func (s *Service) GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error) {
	repo, err := s.repo.GetBySlug(ctx, orgID, providerType, providerBaseURL, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepositoryNotFound
	}
	return repo, nil
}

// FindByOrgSlug looks up a repository by org + slug (ignoring provider).
// Used by AgentFile REPO slug resolution.
func (s *Service) FindByOrgSlug(ctx context.Context, orgID int64, slug string) (*gitprovider.Repository, error) {
	repo, err := s.repo.FindByOrgSlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}
	return repo, nil // nil = not found (no error)
}

// GetCloneURL returns the clone URL for a repository
// Prefers http_clone_url, falls back to clone_url for backward compatibility
func (s *Service) GetCloneURL(ctx context.Context, repoID int64) (string, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return "", err
	}
	if repo.HttpCloneURL != "" {
		return repo.HttpCloneURL, nil
	}
	return repo.CloneURL, nil
}

