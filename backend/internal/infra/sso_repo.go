package infra

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"gorm.io/gorm"
)

type ssoConfigRepository struct{ db *gorm.DB }

var _ sso.Repository = (*ssoConfigRepository)(nil)

// NewSSOConfigRepository creates a new GORM-backed SSO config repository.
func NewSSOConfigRepository(db *gorm.DB) sso.Repository {
	return &ssoConfigRepository{db: db}
}

func (r *ssoConfigRepository) Create(ctx context.Context, cfg *sso.Config) error {
	return r.db.WithContext(ctx).Create(cfg).Error
}

func (r *ssoConfigRepository) GetByID(ctx context.Context, id int64) (*sso.Config, error) {
	var cfg sso.Config
	if err := r.db.WithContext(ctx).First(&cfg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *ssoConfigRepository) GetByDomainAndProtocol(ctx context.Context, domain string, protocol sso.Protocol) (*sso.Config, error) {
	var cfg sso.Config
	err := r.db.WithContext(ctx).
		Where("domain = ? AND protocol = ?", domain, protocol).
		First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *ssoConfigRepository) ListByDomain(ctx context.Context, domain string) ([]*sso.Config, error) {
	var configs []*sso.Config
	err := r.db.WithContext(ctx).Where("domain = ?", domain).Find(&configs).Error
	return configs, err
}

func (r *ssoConfigRepository) GetEnabledByDomain(ctx context.Context, domain string) ([]*sso.Config, error) {
	var configs []*sso.Config
	err := r.db.WithContext(ctx).
		Where("domain = ? AND is_enabled = ?", domain, true).
		Find(&configs).Error
	return configs, err
}

func (r *ssoConfigRepository) List(ctx context.Context, query *sso.ListQuery, offset, limit int) ([]*sso.Config, int64, error) {
	var configs []*sso.Config
	var total int64

	db := r.db.WithContext(ctx).Model(&sso.Config{})

	if query != nil {
		if query.Search != "" {
			// Escape ILIKE wildcards to prevent pattern injection
			escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(query.Search)
			pattern := "%" + escaped + "%"
			db = db.Where("domain ILIKE ? OR name ILIKE ?", pattern, pattern)
		}
		if query.Protocol != "" {
			db = db.Where("protocol = ?", query.Protocol)
		}
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&configs).Error; err != nil {
		return nil, 0, err
	}
	return configs, total, nil
}

func (r *ssoConfigRepository) Update(ctx context.Context, id int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(&sso.Config{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ssoConfigRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&sso.Config{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ssoConfigRepository) HasEnforcedSSO(ctx context.Context, domain string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&sso.Config{}).
		Where("domain = ? AND enforce_sso = ? AND is_enabled = ?", domain, true, true).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
