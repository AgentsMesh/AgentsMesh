package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
)

func (s *WebhookService) RegisterWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, orgSlug string, userID int64) (*WebhookResult, error) {
	result := &WebhookResult{RepoID: repo.ID}

	webhookSecret := generateWebhookSecret()

	webhookURL := s.buildWebhookURL(orgSlug, repo)

	provider, err := s.getGitProviderForUser(ctx, repo, userID)
	if err != nil {
		return s.saveManualSetupConfig(ctx, repo, result, webhookURL, webhookSecret, "Webhook requires manual setup: "+err.Error())
	}

	webhookID, err := provider.RegisterWebhook(ctx, repo.ExternalID, &git.WebhookConfig{
		URL:    webhookURL,
		Secret: webhookSecret,
		Events: []string{"merge_request", "pipeline"},
	})

	if err != nil {
		return s.saveManualSetupConfig(ctx, repo, result, webhookURL, webhookSecret, "Webhook registration failed: "+err.Error())
	}

	return s.saveSuccessConfig(ctx, repo, result, webhookURL, webhookSecret, webhookID)
}

func (s *WebhookService) saveManualSetupConfig(ctx context.Context, repo *gitprovider.Repository, result *WebhookResult, webhookURL, webhookSecret, errorMsg string) (*WebhookResult, error) {
	if s.logger != nil {
		s.logger.Warn("Webhook auto-registration failed, manual setup required",
			"repo_id", repo.ID,
			"repo_slug", repo.Slug,
			"error", errorMsg)
	}

	result.NeedsManualSetup = true
	result.ManualWebhookURL = webhookURL
	result.ManualWebhookSecret = webhookSecret
	result.Error = errorMsg

	now := time.Now().Format(time.RFC3339)
	repo.WebhookConfig = &gitprovider.WebhookConfig{
		URL:              webhookURL,
		Secret:           webhookSecret,
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         false,
		NeedsManualSetup: true,
		LastError:        errorMsg,
		CreatedAt:        now,
	}
	if err := s.repo.Save(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to save webhook config: %w", err)
	}

	return result, nil
}

func (s *WebhookService) saveSuccessConfig(ctx context.Context, repo *gitprovider.Repository, result *WebhookResult, webhookURL, webhookSecret, webhookID string) (*WebhookResult, error) {
	if s.logger != nil {
		s.logger.Info("Webhook registered successfully",
			"repo_id", repo.ID,
			"repo_slug", repo.Slug,
			"webhook_id", webhookID)
	}

	now := time.Now().Format(time.RFC3339)
	repo.WebhookConfig = &gitprovider.WebhookConfig{
		ID:               webhookID,
		URL:              webhookURL,
		Secret:           webhookSecret,
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         true,
		NeedsManualSetup: false,
		CreatedAt:        now,
	}
	if err := s.repo.Save(ctx, repo); err != nil {
		return nil, fmt.Errorf("failed to save webhook config: %w", err)
	}

	result.Registered = true
	result.WebhookID = webhookID
	return result, nil
}

func (s *WebhookService) DeleteWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, userID int64) error {
	if repo.WebhookConfig == nil || repo.WebhookConfig.ID == "" {
		repo.WebhookConfig = nil
		return s.repo.Save(ctx, repo)
	}

	provider, err := s.getGitProviderForUser(ctx, repo, userID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("Cannot delete webhook via API, clearing local config only",
				"repo_id", repo.ID,
				"error", err)
		}
		repo.WebhookConfig = nil
		return s.repo.Save(ctx, repo)
	}

	if err := provider.DeleteWebhook(ctx, repo.ExternalID, repo.WebhookConfig.ID); err != nil {
		if s.logger != nil {
			s.logger.Warn("Failed to delete webhook from provider",
				"repo_id", repo.ID,
				"webhook_id", repo.WebhookConfig.ID,
				"error", err)
		}
	}

	repo.WebhookConfig = nil
	return s.repo.Save(ctx, repo)
}

func (s *WebhookService) buildWebhookURL(orgSlug string, repo *gitprovider.Repository) string {
	return fmt.Sprintf("%s/api/v1/webhooks/%s/%s/%d",
		s.cfg.BaseURL(),
		orgSlug,
		repo.ProviderType,
		repo.ID,
	)
}

func (s *WebhookService) getGitProviderForUser(ctx context.Context, repo *gitprovider.Repository, userID int64) (git.Provider, error) {
	if s.userService == nil {
		return nil, fmt.Errorf("%w: user service not configured", ErrNoAccessToken)
	}

	var accessToken string
	var err error

	accessToken, err = s.userService.GetDecryptedProviderTokenByTypeAndURL(ctx, userID, repo.ProviderType, repo.ProviderBaseURL)
	if err == nil && accessToken != "" {
		provider, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
		if err != nil {
			return nil, err
		}
		return provider, nil
	}

	tokens, err := s.userService.GetDecryptedTokens(ctx, userID, repo.ProviderType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAccessToken, err)
	}

	if tokens.AccessToken == "" {
		return nil, ErrNoAccessToken
	}

	provider, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	return provider, nil
}
