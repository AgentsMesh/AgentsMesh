package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ gitprovider.RepositoryRepo = (*gitProviderRepo)(nil)

type gitProviderRepo struct {
	db *gorm.DB
}

// NewGitProviderRepository creates a new GORM-based gitprovider repository.
func NewGitProviderRepository(db *gorm.DB) gitprovider.RepositoryRepo {
	return &gitProviderRepo{db: db}
}

func (r *gitProviderRepo) FindByOrgAndSlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND provider_type = ? AND provider_base_url = ? AND slug = ? AND deleted_at IS NULL",
			orgID, providerType, providerBaseURL, slug).
		First(&repo).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *gitProviderRepo) Create(ctx context.Context, repo *gitprovider.Repository) error {
	return r.db.WithContext(ctx).Create(repo).Error
}

func (r *gitProviderRepo) GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&repo, id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *gitProviderRepo) Update(ctx context.Context, id int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&gitprovider.Repository{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *gitProviderRepo) CountLoopRefs(ctx context.Context, repoID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM loops WHERE repository_id = ?", repoID).
		Scan(&count).Error
	return count, err
}

func (r *gitProviderRepo) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&gitprovider.Repository{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error
}

func (r *gitProviderRepo) HardDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&gitprovider.Repository{}, id).Error
}

func (r *gitProviderRepo) ListByOrganization(ctx context.Context, orgID int64) ([]*gitprovider.Repository, error) {
	var repos []*gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_active = ? AND deleted_at IS NULL", orgID, true).
		Order("created_at DESC").
		Find(&repos).Error
	return repos, err
}

func (r *gitProviderRepo) ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*gitprovider.Repository, error) {
	var repos []*gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND is_active = ? AND deleted_at IS NULL", orgID, true).
		Where("(visibility = 'organization' OR (visibility = 'private' AND imported_by_user_id = ?))", userID).
		Order("created_at DESC").
		Find(&repos).Error
	return repos, err
}

func (r *gitProviderRepo) GetByExternalID(ctx context.Context, providerType, providerBaseURL, externalID string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("provider_type = ? AND provider_base_url = ? AND external_id = ? AND deleted_at IS NULL",
			providerType, providerBaseURL, externalID).
		First(&repo).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *gitProviderRepo) GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND provider_type = ? AND provider_base_url = ? AND slug = ? AND deleted_at IS NULL",
			orgID, providerType, providerBaseURL, slug).
		First(&repo).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *gitProviderRepo) FindByOrgSlug(ctx context.Context, orgID int64, slug string) (*gitprovider.Repository, error) {
	var repo gitprovider.Repository
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ? AND deleted_at IS NULL", orgID, slug).
		First(&repo).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &repo, nil
}

func (r *gitProviderRepo) GetMaxTicketNumber(ctx context.Context, repoID int64) (int, error) {
	var maxNumber int
	err := r.db.WithContext(ctx).
		Table("tickets").
		Where("repository_id = ?", repoID).
		Select("COALESCE(MAX(number), 0)").
		Scan(&maxNumber).Error
	return maxNumber, err
}

func (r *gitProviderRepo) Save(ctx context.Context, repo *gitprovider.Repository) error {
	return r.db.WithContext(ctx).Save(repo).Error
}

func (r *gitProviderRepo) ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]gitprovider.MergeRequestRow, error) {
	query := r.db.WithContext(ctx).
		Model(&ticket.MergeRequest{}).
		Where("repository_id = ?", repoID)

	if branch != "" {
		query = query.Where("source_branch = ?", branch)
	}
	if state != "" && state != "all" {
		query = query.Where("state = ?", state)
	}
	query = query.Order("created_at DESC")

	var mrs []ticket.MergeRequest
	if err := query.Find(&mrs).Error; err != nil {
		return nil, err
	}

	rows := make([]gitprovider.MergeRequestRow, 0, len(mrs))
	for _, mr := range mrs {
		rows = append(rows, gitprovider.MergeRequestRow{
			ID:             mr.ID,
			MRIID:          mr.MRIID,
			Title:          mr.Title,
			State:          mr.State,
			MRURL:          mr.MRURL,
			SourceBranch:   mr.SourceBranch,
			TargetBranch:   mr.TargetBranch,
			PipelineStatus: mr.PipelineStatus,
			PipelineID:     mr.PipelineID,
			PipelineURL:    mr.PipelineURL,
			TicketID:       mr.TicketID,
			PodID:          mr.PodID,
		})
	}
	return rows, nil
}
