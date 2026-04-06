package repository

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
)

// Create creates a new repository configuration.
// If the same repository already exists, it updates provider metadata
// (idempotent import) so that re-importing after a provider reconnect
// does not fail.
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*gitprovider.Repository, error) {
	// Check if repository already exists (unique: org + provider_type + provider_base_url + slug)
	existing, err := s.repo.FindByOrgAndSlug(ctx, req.OrganizationID, req.ProviderType, req.ProviderBaseURL, req.Slug)
	if err != nil {
		return nil, err
	}

	// Idempotent import: update provider-sourced metadata, preserve user-configured fields
	if existing != nil {
		return s.handleIdempotentImport(ctx, existing, req)
	}

	return s.createNewRepo(ctx, req)
}

// handleIdempotentImport updates provider-sourced metadata for an existing repository
func (s *Service) handleIdempotentImport(ctx context.Context, existing *gitprovider.Repository, req *CreateRequest) (*gitprovider.Repository, error) {
	updates := map[string]interface{}{
		"name":        req.Name,
		"external_id": req.ExternalID,
		"is_active":   true,
	}
	if req.DefaultBranch != "" {
		updates["default_branch"] = req.DefaultBranch
	}
	if req.ImportedByUserID != nil {
		updates["imported_by_user_id"] = *req.ImportedByUserID
	}
	if req.HttpCloneURL != "" {
		updates["http_clone_url"] = req.HttpCloneURL
	}
	if req.SshCloneURL != "" {
		updates["ssh_clone_url"] = req.SshCloneURL
	}
	return s.Update(ctx, existing.ID, updates)
}

// createNewRepo creates a new repository record
func (s *Service) createNewRepo(ctx context.Context, req *CreateRequest) (*gitprovider.Repository, error) {
	repo := &gitprovider.Repository{
		OrganizationID:   req.OrganizationID,
		ProviderType:     req.ProviderType,
		ProviderBaseURL:  req.ProviderBaseURL,
		HttpCloneURL:     req.HttpCloneURL,
		SshCloneURL:      req.SshCloneURL,
		ExternalID:       req.ExternalID,
		Name:             req.Name,
		Slug:             req.Slug,
		DefaultBranch:    req.DefaultBranch,
		TicketPrefix:     req.TicketPrefix,
		Visibility:       req.Visibility,
		ImportedByUserID: req.ImportedByUserID,
		IsActive:         true,
	}

	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}
	if repo.Visibility == "" {
		repo.Visibility = "organization"
	}

	// Generate clone URLs if not provided
	if repo.HttpCloneURL == "" || repo.SshCloneURL == "" {
		httpURL, sshURL := generateCloneURLs(repo.ProviderType, repo.ProviderBaseURL, repo.Slug)
		if repo.HttpCloneURL == "" {
			repo.HttpCloneURL = httpURL
		}
		if repo.SshCloneURL == "" {
			repo.SshCloneURL = sshURL
		}
	}

	if err := s.repo.Create(ctx, repo); err != nil {
		slog.Error("failed to create repository", "org_id", req.OrganizationID, "slug", req.Slug, "error", err)
		return nil, err
	}

	slog.Info("repository created", "repo_id", repo.ID, "org_id", req.OrganizationID, "slug", req.Slug)
	return repo, nil
}

// CreateWithWebhook creates a repository and registers a webhook
// orgSlug is required for building the webhook URL
func (s *Service) CreateWithWebhook(ctx context.Context, req *CreateRequest, orgSlug string) (*gitprovider.Repository, *WebhookResult, error) {
	repo, err := s.Create(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	// If webhook service is configured and user ID is available, try to register webhook
	var webhookResult *WebhookResult
	if s.webhookService != nil && req.ImportedByUserID != nil {
		// Register webhook asynchronously to not block repository creation
		go func() {
			bgCtx := context.Background()
			result, err := s.webhookService.RegisterWebhookForRepository(bgCtx, repo, orgSlug, *req.ImportedByUserID)
			if err != nil {
				// Log error but don't fail - webhook can be registered manually later
				if s.webhookService.logger != nil {
					s.webhookService.logger.Error("Failed to register webhook during repository creation",
						"repo_id", repo.ID,
						"error", err)
				}
			} else if result.NeedsManualSetup {
				if s.webhookService.logger != nil {
					s.webhookService.logger.Info("Webhook requires manual setup",
						"repo_id", repo.ID,
						"webhook_url", result.ManualWebhookURL)
				}
			}
		}()

		// Return a placeholder result indicating webhook registration is in progress
		webhookResult = &WebhookResult{
			RepoID: repo.ID,
			Error:  "Webhook registration in progress",
		}
	}

	return repo, webhookResult, nil
}

// generateCloneURLs generates both HTTP and SSH clone URLs based on provider type
func generateCloneURLs(providerType, baseURL, slug string) (httpURL, sshURL string) {
	switch providerType {
	case "github":
		httpURL = "https://github.com/" + slug + ".git"
		sshURL = "git@github.com:" + slug + ".git"
	case "gitlab":
		httpURL = baseURL + "/" + slug + ".git"
		host := extractHost(baseURL)
		sshURL = "git@" + host + ":" + slug + ".git"
	case "gitee":
		httpURL = "https://gitee.com/" + slug + ".git"
		sshURL = "git@gitee.com:" + slug + ".git"
	default:
		httpURL = baseURL + "/" + slug + ".git"
		host := extractHost(baseURL)
		sshURL = "git@" + host + ":" + slug + ".git"
	}
	return
}

// extractHost extracts the host from a URL (e.g., "https://gitlab.company.com" -> "gitlab.company.com")
func extractHost(baseURL string) string {
	host := baseURL
	// Remove protocol prefix
	for _, prefix := range []string{"https://", "http://"} {
		if len(host) > len(prefix) && host[:len(prefix)] == prefix {
			host = host[len(prefix):]
			break
		}
	}
	// Remove trailing slash
	if len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}
	return host
}
