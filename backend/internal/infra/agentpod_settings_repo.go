package infra

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

var _ agentpod.SettingsRepository = (*settingsRepo)(nil)

type settingsRepo struct{ db *gorm.DB }

// NewSettingsRepository creates a new SettingsRepository backed by GORM.
func NewSettingsRepository(db *gorm.DB) agentpod.SettingsRepository {
	return &settingsRepo{db: db}
}

func (r *settingsRepo) GetByUserID(ctx context.Context, userID int64) (*agentpod.UserAgentPodSettings, error) {
	var settings agentpod.UserAgentPodSettings
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &settings, nil
}

func (r *settingsRepo) Create(ctx context.Context, settings *agentpod.UserAgentPodSettings) error {
	return r.db.WithContext(ctx).Create(settings).Error
}

func (r *settingsRepo) Save(ctx context.Context, settings *agentpod.UserAgentPodSettings) error {
	return r.db.WithContext(ctx).Save(settings).Error
}

func (r *settingsRepo) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&agentpod.UserAgentPodSettings{}).Error
}
