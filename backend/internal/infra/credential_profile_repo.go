package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ agent.CredentialProfileRepository = (*credentialProfileRepo)(nil)

type credentialProfileRepo struct {
	db *gorm.DB
}

// NewCredentialProfileRepository creates a new GORM-based credential profile repository
func NewCredentialProfileRepository(db *gorm.DB) agent.CredentialProfileRepository {
	return &credentialProfileRepo{db: db}
}

func (r *credentialProfileRepo) Create(ctx context.Context, profile *agent.UserAgentCredentialProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *credentialProfileRepo) GetWithAgent(ctx context.Context, userID, profileID int64) (*agent.UserAgentCredentialProfile, error) {
	var profile agent.UserAgentCredentialProfile
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("id = ? AND user_id = ?", profileID, userID).
		First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *credentialProfileRepo) GetByName(ctx context.Context, userID int64, agentSlug, name string) (*agent.UserAgentCredentialProfile, error) {
	var profile agent.UserAgentCredentialProfile
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ? AND agent_slug = ? AND name = ? AND is_active = ?", userID, agentSlug, name, true).
		First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *credentialProfileRepo) Delete(ctx context.Context, userID, profileID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", profileID, userID).
		Delete(&agent.UserAgentCredentialProfile{})
	return result.RowsAffected, result.Error
}

func (r *credentialProfileRepo) ListActiveWithAgent(ctx context.Context, userID int64) ([]*agent.UserAgentCredentialProfile, error) {
	var profiles []*agent.UserAgentCredentialProfile
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("agent_slug, is_default DESC, name").
		Find(&profiles).Error
	return profiles, err
}

func (r *credentialProfileRepo) ListByAgentSlug(ctx context.Context, userID int64, agentSlug string) ([]*agent.UserAgentCredentialProfile, error) {
	var profiles []*agent.UserAgentCredentialProfile
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ? AND agent_slug = ? AND is_active = ?", userID, agentSlug, true).
		Order("is_default DESC, name").
		Find(&profiles).Error
	return profiles, err
}

func (r *credentialProfileRepo) GetDefault(ctx context.Context, userID int64, agentSlug string) (*agent.UserAgentCredentialProfile, error) {
	var profile agent.UserAgentCredentialProfile
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ? AND agent_slug = ? AND is_default = ? AND is_active = ?", userID, agentSlug, true, true).
		First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (r *credentialProfileRepo) NameExists(ctx context.Context, userID int64, agentSlug string, name string, excludeID *int64) (bool, error) {
	query := r.db.WithContext(ctx).Model(&agent.UserAgentCredentialProfile{}).
		Where("user_id = ? AND agent_slug = ? AND name = ?", userID, agentSlug, name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *credentialProfileRepo) UnsetDefaults(ctx context.Context, userID int64, agentSlug string) error {
	return r.db.WithContext(ctx).Model(&agent.UserAgentCredentialProfile{}).
		Where("user_id = ? AND agent_slug = ?", userID, agentSlug).
		Update("is_default", false).Error
}

func (r *credentialProfileRepo) Update(ctx context.Context, profile *agent.UserAgentCredentialProfile, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(profile).Updates(updates).Error
}

func (r *credentialProfileRepo) SetDefault(ctx context.Context, profile *agent.UserAgentCredentialProfile) error {
	return r.db.WithContext(ctx).Model(profile).Update("is_default", true).Error
}
