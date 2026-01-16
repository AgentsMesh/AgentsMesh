package repository

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"gorm.io/gorm"
)

var (
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrRepositoryExists   = errors.New("repository already exists")
	ErrNoPermission       = errors.New("no permission to access this repository")
)

// Service handles repository operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new repository service
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// CreateRequest represents repository creation request
// Self-contained: no git_provider_id, includes all necessary info
type CreateRequest struct {
	OrganizationID   int64
	ProviderType     string // github, gitlab, gitee, generic
	ProviderBaseURL  string // https://github.com, https://gitlab.company.com
	CloneURL         string // Full clone URL
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
	var existing gitprovider.Repository
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND provider_type = ? AND provider_base_url = ? AND full_path = ?",
			req.OrganizationID, req.ProviderType, req.ProviderBaseURL, req.FullPath).
		First(&existing).Error; err == nil {
		return nil, ErrRepositoryExists
	}

	repo := &gitprovider.Repository{
		OrganizationID:   req.OrganizationID,
		ProviderType:     req.ProviderType,
		ProviderBaseURL:  req.ProviderBaseURL,
		CloneURL:         req.CloneURL,
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

	// Generate clone URL if not provided
	if repo.CloneURL == "" {
		repo.CloneURL = generateCloneURL(repo.ProviderType, repo.ProviderBaseURL, repo.FullPath)
	}

	if err := s.db.WithContext(ctx).Create(repo).Error; err != nil {
		return nil, err
	}

	return repo, nil
}

// generateCloneURL generates clone URL based on provider type
func generateCloneURL(providerType, baseURL, fullPath string) string {
	switch providerType {
	case "github":
		return "https://github.com/" + fullPath + ".git"
	case "gitlab":
		return baseURL + "/" + fullPath + ".git"
	case "gitee":
		return "https://gitee.com/" + fullPath + ".git"
	default:
		return baseURL + "/" + fullPath + ".git"
	}
}

// GetByID returns a repository by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	if err := s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&repo, id).Error; err != nil {
		return nil, ErrRepositoryNotFound
	}
	return &repo, nil
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
	if err := s.db.WithContext(ctx).Model(&gitprovider.Repository{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete soft deletes a repository
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Model(&gitprovider.Repository{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

// HardDelete permanently deletes a repository
func (s *Service) HardDelete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Unscoped().Delete(&gitprovider.Repository{}, id).Error
}

// ListByOrganization returns repositories for an organization
func (s *Service) ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error) {
	var repos []*gitprovider.Repository
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND is_active = ? AND deleted_at IS NULL", orgID, true).
		Order("created_at DESC").Find(&repos).Error
	return repos, err
}

// ListByOrganizationForUser returns repositories visible to a specific user
func (s *Service) ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error) {
	var repos []*gitprovider.Repository
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND is_active = ? AND deleted_at IS NULL", orgID, true).
		Where("(visibility = 'organization' OR (visibility = 'private' AND imported_by_user_id = ?))", userID).
		Order("created_at DESC").Find(&repos).Error
	return repos, err
}

// GetByExternalID returns a repository by provider type, base URL, and external ID
func (s *Service) GetByExternalID(ctx context.Context, providerType, providerBaseURL, externalID string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	if err := s.db.WithContext(ctx).
		Where("provider_type = ? AND provider_base_url = ? AND external_id = ? AND deleted_at IS NULL",
			providerType, providerBaseURL, externalID).
		First(&repo).Error; err != nil {
		return nil, ErrRepositoryNotFound
	}
	return &repo, nil
}

// GetByFullPath returns a repository by organization, provider, and full path
func (s *Service) GetByFullPath(ctx context.Context, orgID int64, providerType, providerBaseURL, fullPath string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND provider_type = ? AND provider_base_url = ? AND full_path = ? AND deleted_at IS NULL",
			orgID, providerType, providerBaseURL, fullPath).
		First(&repo).Error; err != nil {
		return nil, ErrRepositoryNotFound
	}
	return &repo, nil
}

// GetCloneURL returns the clone URL for a repository
func (s *Service) GetCloneURL(ctx context.Context, repoID int64) (string, error) {
	repo, err := s.GetByID(ctx, repoID)
	if err != nil {
		return "", err
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
	var maxNumber int
	s.db.WithContext(ctx).
		Table("tickets").
		Where("repository_id = ?", repoID).
		Select("COALESCE(MAX(number), 0)").
		Scan(&maxNumber)
	return maxNumber + 1, nil
}
