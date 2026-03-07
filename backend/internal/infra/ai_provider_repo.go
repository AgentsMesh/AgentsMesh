package infra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

var _ agentpod.AIProviderRepository = (*aiProviderRepo)(nil)

type aiProviderRepo struct{ db *gorm.DB }

// NewAIProviderRepository creates a new AIProviderRepository backed by GORM.
func NewAIProviderRepository(db *gorm.DB) agentpod.AIProviderRepository {
	return &aiProviderRepo{db: db}
}

func (r *aiProviderRepo) GetDefaultByType(ctx context.Context, userID int64, providerType string) (*agentpod.UserAIProvider, error) {
	var provider agentpod.UserAIProvider
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ? AND is_default = ? AND is_enabled = ?",
			userID, providerType, true, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &provider, nil
}

func (r *aiProviderRepo) GetEnabledByID(ctx context.Context, providerID int64) (*agentpod.UserAIProvider, error) {
	var provider agentpod.UserAIProvider
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_enabled = ?", providerID, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (r *aiProviderRepo) GetByID(ctx context.Context, providerID int64) (*agentpod.UserAIProvider, error) {
	var provider agentpod.UserAIProvider
	err := r.db.WithContext(ctx).First(&provider, providerID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func (r *aiProviderRepo) ListByUser(ctx context.Context, userID int64) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider_type, name").Find(&providers).Error
	return providers, err
}

func (r *aiProviderRepo) ListByUserAndType(ctx context.Context, userID int64, providerType string) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Order("is_default DESC, name").Find(&providers).Error
	return providers, err
}

func (r *aiProviderRepo) Create(ctx context.Context, provider *agentpod.UserAIProvider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *aiProviderRepo) Save(ctx context.Context, provider *agentpod.UserAIProvider) error {
	return r.db.WithContext(ctx).Save(provider).Error
}

func (r *aiProviderRepo) Delete(ctx context.Context, providerID int64) error {
	return r.db.WithContext(ctx).Delete(&agentpod.UserAIProvider{}, providerID).Error
}

func (r *aiProviderRepo) SetDefault(ctx context.Context, providerID int64) error {
	return r.db.WithContext(ctx).Model(&agentpod.UserAIProvider{}).
		Where("id = ?", providerID).Update("is_default", true).Error
}

func (r *aiProviderRepo) ClearDefaults(ctx context.Context, userID int64, providerType string) error {
	return r.db.WithContext(ctx).
		Model(&agentpod.UserAIProvider{}).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Update("is_default", false).Error
}
