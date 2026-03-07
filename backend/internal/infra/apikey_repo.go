package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ apikey.Repository = (*apikeyRepo)(nil)

type apikeyRepo struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new GORM-based API key repository
func NewAPIKeyRepository(db *gorm.DB) apikey.Repository {
	return &apikeyRepo{db: db}
}

func (r *apikeyRepo) Create(ctx context.Context, key *apikey.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *apikeyRepo) GetByID(ctx context.Context, id int64, orgID int64) (*apikey.APIKey, error) {
	var key apikey.APIKey
	err := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, orgID).
		First(&key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apikey.ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *apikeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
	var key apikey.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apikey.ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *apikeyRepo) List(ctx context.Context, orgID int64, isEnabled *bool, limit, offset int) ([]apikey.APIKey, int64, error) {
	var keys []apikey.APIKey
	var total int64

	query := r.db.WithContext(ctx).Model(&apikey.APIKey{}).
		Where("organization_id = ?", orgID)

	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&keys).Error; err != nil {
		return nil, 0, err
	}

	return keys, total, nil
}

func (r *apikeyRepo) Update(ctx context.Context, key *apikey.APIKey, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(key).Updates(updates).Error
}

func (r *apikeyRepo) Delete(ctx context.Context, key *apikey.APIKey) error {
	return r.db.WithContext(ctx).Delete(key).Error
}

func (r *apikeyRepo) UpdateLastUsed(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&apikey.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *apikeyRepo) CheckDuplicateName(ctx context.Context, orgID int64, name string, excludeID *int64) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&apikey.APIKey{}).
		Where("organization_id = ? AND name = ?", orgID, name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
