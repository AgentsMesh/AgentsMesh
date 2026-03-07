package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
)

// RegisterWebhookForRepository registers a webhook for a repository.
// If automatic registration fails (no OAuth token, permission denied, etc.),
// it saves the webhook config with NeedsManualSetup=true.
func (s *WebhookService) RegisterWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, orgSlug string, userID int64) (*WebhookResult, error) {
	result := &WebhookResult{RepoID: repo.ID}

	// Generate repository-specific webhook secret
	webhookSecret := generateWebhookSecret()

	// Build webhook URL with org_slug and repo_id
	webhookURL := s.buildWebhookURL(orgSlug, repo)

	// Try to get user's git provider and access token
	provider, err := s.getGitProviderForUser(ctx, repo, userID)
	if err != nil {
		return s.saveManualSetupConfig(ctx, repo, result, webhookURL, webhookSecret, "Webhook requires manual setup: "+err.Error())
	}

	// Try to register webhook via git provider API
	webhookID, err := provider.RegisterWebhook(ctx, repo.ExternalID, &git.WebhookConfig{
		URL:    webhookURL,
		Secret: webhookSecret,
		Events: []string{"merge_request", "pipeline"},
	})

	if err != nil {
		return s.saveManualSetupConfig(ctx, repo, result, webhookURL, webhookSecret, "Webhook registration failed: "+err.Error())
	}

	// Registration successful
	return s.saveSuccessConfig(ctx, repo, result, webhookURL, webhookSecret, webhookID)
}

// saveManualSetupConfig saves webhook config for manual setup when auto-registration fails
func (s *WebhookService) saveManualSetupConfig(ctx context.Context, repo *gitprovider.Repository, result *WebhookResult, webhookURL, webhookSecret, errorMsg string) (*WebhookResult, error) {
	if s.logger != nil {
		s.logger.Warn("Webhook auto-registration failed, manual setup required",
			"repo_id", repo.ID,
			"repo_full_path", repo.FullPath,
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

// saveSuccessConfig saves webhook config after successful auto-registration
func (s *WebhookService) saveSuccessConfig(ctx context.Context, repo *gitprovider.Repository, result *WebhookResult, webhookURL, webhookSecret, webhookID string) (*WebhookResult, error) {
	if s.logger != nil {
		s.logger.Info("Webhook registered successfully",
			"repo_id", repo.ID,
			"repo_full_path", repo.FullPath,
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

// DeleteWebhookForRepository deletes a webhook from the git provider
func (s *WebhookService) DeleteWebhookForRepository(ctx context.Context, repo *gitprovider.Repository, userID int64) error {
	if repo.WebhookConfig == nil || repo.WebhookConfig.ID == "" {
		// No webhook registered - just clear the config
		repo.WebhookConfig = nil
		return s.repo.Save(ctx, repo)
	}

	// Try to get user's git provider
	provider, err := s.getGitProviderForUser(ctx, repo, userID)
	if err != nil {
		// Can't delete via API - just clear the local config
		if s.logger != nil {
			s.logger.Warn("Cannot delete webhook via API, clearing local config only",
				"repo_id", repo.ID,
				"error", err)
		}
		repo.WebhookConfig = nil
		return s.repo.Save(ctx, repo)
	}

	// Try to delete webhook from git provider
	if err := provider.DeleteWebhook(ctx, repo.ExternalID, repo.WebhookConfig.ID); err != nil {
		if s.logger != nil {
			s.logger.Warn("Failed to delete webhook from provider",
				"repo_id", repo.ID,
				"webhook_id", repo.WebhookConfig.ID,
				"error", err)
		}
		// Continue anyway - clear local config
	}

	repo.WebhookConfig = nil
	return s.repo.Save(ctx, repo)
}

// buildWebhookURL constructs the webhook URL for a repository
// Format: {BaseURL}/api/v1/webhooks/{org_slug}/{provider_type}/{repo_id}
func (s *WebhookService) buildWebhookURL(orgSlug string, repo *gitprovider.Repository) string {
	return fmt.Sprintf("%s/api/v1/webhooks/%s/%s/%d",
		s.cfg.BaseURL(),
		orgSlug,
		repo.ProviderType,
		repo.ID,
	)
}

// getGitProviderForUser returns a git provider instance using the user's access token
// It first tries to get token from UserRepositoryProvider (bot token or linked OAuth identity),
// then falls back to OAuth tokens
func (s *WebhookService) getGitProviderForUser(ctx context.Context, repo *gitprovider.Repository, userID int64) (git.Provider, error) {
	// Check if userService is available
	if s.userService == nil {
		return nil, fmt.Errorf("%w: user service not configured", ErrNoAccessToken)
	}

	var accessToken string
	var err error

	// 1. Try to get token from UserRepositoryProvider (bot token or linked OAuth identity)
	accessToken, err = s.userService.GetDecryptedProviderTokenByTypeAndURL(ctx, userID, repo.ProviderType, repo.ProviderBaseURL)
	if err == nil && accessToken != "" {
		// Found token from repository provider
		provider, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
		if err != nil {
			return nil, err
		}
		return provider, nil
	}

	// 2. Fall back to OAuth tokens from user_identities
	// Map repository provider type to OAuth provider name
	tokens, err := s.userService.GetDecryptedTokens(ctx, userID, repo.ProviderType)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAccessToken, err)
	}

	if tokens.AccessToken == "" {
		return nil, ErrNoAccessToken
	}

	// Create git provider client
	provider, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, tokens.AccessToken)
	if err != nil {
		return nil, err
	}

	return provider, nil
}
