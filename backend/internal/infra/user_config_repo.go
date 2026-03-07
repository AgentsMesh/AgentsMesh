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

func (r *userConfigRepo) GetByUserAndAgentType(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentConfig, error) {
	var config agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *userConfigRepo) Upsert(ctx context.Context, userID, agentTypeID int64, configValues agent.ConfigValues) error {
	var existing agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&existing).Error

	if err != nil {
		// Record doesn't exist, create new one
		config := &agent.UserAgentConfig{
			UserID:       userID,
			AgentTypeID:  agentTypeID,
			ConfigValues: configValues,
		}
		return r.db.WithContext(ctx).Create(config).Error
	}

	// Record exists, update config_values
	return r.db.WithContext(ctx).
		Model(&existing).
		Update("config_values", configValues).Error
}

func (r *userConfigRepo) Delete(ctx context.Context, userID, agentTypeID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Delete(&agent.UserAgentConfig{}).Error
}

func (r *userConfigRepo) ListByUser(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	var configs []*agent.UserAgentConfig
	err := r.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}
