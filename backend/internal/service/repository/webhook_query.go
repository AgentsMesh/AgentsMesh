package repository

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

// GetWebhookStatus returns the webhook status for a repository
func (s *WebhookService) GetWebhookStatus(ctx context.Context, repo *gitprovider.Repository) *gitprovider.WebhookStatus {
	if repo.WebhookConfig == nil {
		return &gitprovider.WebhookStatus{Registered: false}
	}
	return repo.WebhookConfig.ToStatus()
}

// GetWebhookSecret returns the webhook secret for manual configuration.
// Only returns the secret if NeedsManualSetup is true.
func (s *WebhookService) GetWebhookSecret(ctx context.Context, repo *gitprovider.Repository) (string, error) {
	if repo.WebhookConfig == nil {
		return "", ErrWebhookNotFound
	}
	if !repo.WebhookConfig.NeedsManualSetup {
		return "", errors.New("webhook is already automatically registered, no manual setup required")
	}
	return repo.WebhookConfig.Secret, nil
}

// MarkWebhookAsConfigured marks a webhook as manually configured
func (s *WebhookService) MarkWebhookAsConfigured(ctx context.Context, repo *gitprovider.Repository) error {
	if repo.WebhookConfig == nil {
		return ErrWebhookNotFound
	}

	repo.WebhookConfig.IsActive = true
	repo.WebhookConfig.NeedsManualSetup = false
	repo.WebhookConfig.LastError = ""

	return s.repo.Save(ctx, repo)
}

// VerifyWebhookSecret verifies that the provided secret matches the repository's webhook secret
func (s *WebhookService) VerifyWebhookSecret(ctx context.Context, repoID int64, providedSecret string) (bool, error) {
	repo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return false, err
	}
	if repo == nil {
		return false, ErrRepositoryNotFound
	}

	if repo.WebhookConfig == nil || repo.WebhookConfig.Secret == "" {
		return false, ErrWebhookNotFound
	}

	return repo.WebhookConfig.Secret == providedSecret, nil
}

// GetRepositoryByIDWithWebhook retrieves a repository by ID with webhook config
func (s *WebhookService) GetRepositoryByIDWithWebhook(ctx context.Context, repoID int64) (*gitprovider.Repository, error) {
	repo, err := s.repo.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepositoryNotFound
	}
	return repo, nil
}
