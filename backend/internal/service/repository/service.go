package repository

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
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
	FullPath         string
	DefaultBranch    string
	TicketPrefix     *string
	Visibility       string // "organization" or "private"
	ImportedByUserID *int64 // User who imported this repo
}

// Create creates a new repository configuration
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*gitprovider.Repository, error) {
	// Check if repository already exists (unique: org + provider_type + provider_base_url + full_path)
	existing, err := s.repo.FindByOrgAndPath(ctx, req.OrganizationID, req.ProviderType, req.ProviderBaseURL, req.FullPath)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRepositoryExists
	}

	repo := &gitprovider.Repository{
		OrganizationID:   req.OrganizationID,
		ProviderType:     req.ProviderType,
		ProviderBaseURL:  req.ProviderBaseURL,
		CloneURL:         req.CloneURL,
		HttpCloneURL:     req.HttpCloneURL,
		SshCloneURL:      req.SshCloneURL,
		ExternalID:       req.ExternalID,
		Name:             req.Name,
		FullPath:         req.FullPath,
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
		httpURL, sshURL := generateCloneURLs(repo.ProviderType, repo.ProviderBaseURL, repo.FullPath)
		if repo.HttpCloneURL == "" {
			repo.HttpCloneURL = httpURL
		}
		if repo.SshCloneURL == "" {
			repo.SshCloneURL = sshURL
		}
	}

	// Keep legacy clone_url populated for backwards compatibility
	if repo.CloneURL == "" {
		repo.CloneURL = repo.HttpCloneURL
	}

	if err := s.repo.Create(ctx, repo); err != nil {
		return nil, err
	}

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
func generateCloneURLs(providerType, baseURL, fullPath string) (httpURL, sshURL string) {
	switch providerType {
	case "github":
		httpURL = "https://github.com/" + fullPath + ".git"
		sshURL = "git@github.com:" + fullPath + ".git"
	case "gitlab":
		httpURL = baseURL + "/" + fullPath + ".git"
		host := extractHost(baseURL)
		sshURL = "git@" + host + ":" + fullPath + ".git"
	case "gitee":
		httpURL = "https://gitee.com/" + fullPath + ".git"
		sshURL = "git@gitee.com:" + fullPath + ".git"
	default:
		httpURL = baseURL + "/" + fullPath + ".git"
		host := extractHost(baseURL)
		sshURL = "git@" + host + ":" + fullPath + ".git"
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
		return nil, err
	}
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
	return s.repo.SoftDelete(ctx, id)
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
	return s.repo.HardDelete(ctx, id)
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

// GetByFullPath returns a repository by organization, provider, and full path
func (s *Service) GetByFullPath(ctx context.Context, orgID int64, providerType, providerBaseURL, fullPath string) (*gitprovider.Repository, error) {
	repo, err := s.repo.GetByFullPath(ctx, orgID, providerType, providerBaseURL, fullPath)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepositoryNotFound
	}
	return repo, nil
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

// SyncFromProvider syncs repository info from git provider using user's token
func (s *Service) SyncFromProvider(ctx context.Context, repoID int64, accessToken string) (*gitprovider.Repository, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	// Create git provider client using repo's self-contained info
	client, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
	if err != nil {
		return nil, err
	}

	project, err := client.GetProject(ctx, repo.ExternalID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":           project.Name,
		"full_path":      project.FullPath,
		"default_branch": project.DefaultBranch,
	}
	if project.CloneURL != "" {
		updates["clone_url"] = project.CloneURL
		updates["http_clone_url"] = project.CloneURL
	}
	if project.SSHCloneURL != "" {
		updates["ssh_clone_url"] = project.SSHCloneURL
	}

	return s.Update(ctx, repoID, updates)
}

// ListBranches lists branches for a repository using user's token
func (s *Service) ListBranches(ctx context.Context, repoID int64, accessToken string) ([]string, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}

	// Create git provider client using repo's self-contained info
	client, err := git.NewProvider(repo.ProviderType, repo.ProviderBaseURL, accessToken)
	if err != nil {
		return nil, err
	}

	branches, err := client.ListBranches(ctx, repo.ExternalID)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, nil
}

// GetNextTicketNumber returns the next ticket number for a repository
func (s *Service) GetNextTicketNumber(ctx context.Context, repoID int64) (int, error) {
	maxNumber, err := s.repo.GetMaxTicketNumber(ctx, repoID)
	if err != nil {
		return 0, err
	}
	return maxNumber + 1, nil
}
