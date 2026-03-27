package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ agent.UserConfigRepository = (*userConfigRepo)(nil)

type userConfigRepo struct {
	db *gorm.DB
}

// NewUserConfigRepository creates a new GORM-based user config repository
func NewUserConfigRepository(db *gorm.DB) agent.UserConfigRepository {
	return &userConfigRepo{db: db}
}

func (r *userConfigRepo) GetByUserAndAgentSlug(ctx context.Context, userID int64, agentSlug string) (*agent.UserAgentConfig, error) {
	var config agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ? AND agent_slug = ?", userID, agentSlug).
		First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *userConfigRepo) Upsert(ctx context.Context, userID int64, agentSlug string, configValues agent.ConfigValues) error {
	var existing agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND agent_slug = ?", userID, agentSlug).
		First(&existing).Error

	if err != nil {
		// Record doesn't exist, create new one
		config := &agent.UserAgentConfig{
			UserID:       userID,
			AgentSlug:  agentSlug,
			ConfigValues: configValues,
		}
		return r.db.WithContext(ctx).Create(config).Error
	}

	// Record exists, update config_values
	return r.db.WithContext(ctx).
		Model(&existing).
		Update("config_values", configValues).Error
}

func (r *userConfigRepo) Delete(ctx context.Context, userID int64, agentSlug string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND agent_slug = ?", userID, agentSlug).
		Delete(&agent.UserAgentConfig{}).Error
}

func (r *userConfigRepo) ListByUser(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	var configs []*agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}
