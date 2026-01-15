package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// Errors for UserConfigService
var (
	ErrUserAgentConfigNotFound = errors.New("user agent config not found")
)

// UserConfigService handles user personal runtime configuration
type UserConfigService struct {
	db               *gorm.DB
	agentTypeService AgentTypeProvider
}

// NewUserConfigService creates a new user config service
func NewUserConfigService(db *gorm.DB, agentTypeService AgentTypeProvider) *UserConfigService {
	return &UserConfigService{
		db:               db,
		agentTypeService: agentTypeService,
	}
}

// GetUserAgentConfig returns the user's personal config for an agent type
func (s *UserConfigService) GetUserAgentConfig(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentConfig, error) {
	var config agent.UserAgentConfig
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty config if not found
			return &agent.UserAgentConfig{
				UserID:       userID,
				AgentTypeID:  agentTypeID,
				ConfigValues: make(agent.ConfigValues),
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// SetUserAgentConfig sets the user's personal config for an agent type
func (s *UserConfigService) SetUserAgentConfig(ctx context.Context, userID, agentTypeID int64, configValues agent.ConfigValues) (*agent.UserAgentConfig, error) {
	// Verify agent type exists
	if _, err := s.agentTypeService.GetAgentType(ctx, agentTypeID); err != nil {
		return nil, err
	}

	// Try to find existing config
	var existing agent.UserAgentConfig
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		First(&existing).Error

	if err != nil {
		// Record doesn't exist, create new one
		config := &agent.UserAgentConfig{
			UserID:       userID,
			AgentTypeID:  agentTypeID,
			ConfigValues: configValues,
		}
		if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
			return nil, err
		}
	} else {
		// Record exists, update config_values explicitly
		err = s.db.WithContext(ctx).
			Model(&existing).
			Update("config_values", configValues).Error
		if err != nil {
			return nil, err
		}
	}

	return s.GetUserAgentConfig(ctx, userID, agentTypeID)
}

// DeleteUserAgentConfig deletes the user's personal config for an agent type
func (s *UserConfigService) DeleteUserAgentConfig(ctx context.Context, userID, agentTypeID int64) error {
	result := s.db.WithContext(ctx).
		Where("user_id = ? AND agent_type_id = ?", userID, agentTypeID).
		Delete(&agent.UserAgentConfig{})
	if result.Error != nil {
		return result.Error
	}
	// Not treating "no rows affected" as an error - it's fine if there was nothing to delete
	return nil
}

// ListUserAgentConfigs returns all personal configs for a user
func (s *UserConfigService) ListUserAgentConfigs(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	var configs []*agent.UserAgentConfig
	err := s.db.WithContext(ctx).
		Preload("AgentType").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}

// GetUserEffectiveConfig returns the effective config by merging ConfigSchema defaults and user personal config
func (s *UserConfigService) GetUserEffectiveConfig(ctx context.Context, userID, agentTypeID int64, overrides agent.ConfigValues) agent.ConfigValues {
	result := make(agent.ConfigValues)

	// 1. Get ConfigSchema defaults from AgentType
	agentType, err := s.agentTypeService.GetAgentType(ctx, agentTypeID)
	if err == nil && agentType.ConfigSchema.Fields != nil {
		for _, field := range agentType.ConfigSchema.Fields {
			if field.Default != nil {
				result[field.Name] = field.Default
			}
		}
	}

	// 2. Get user's personal config
	userConfig, err := s.GetUserAgentConfig(ctx, userID, agentTypeID)
	if err == nil && userConfig.ConfigValues != nil {
		result = agent.MergeConfigs(result, userConfig.ConfigValues)
	}

	// 3. Apply overrides (from CreatePod request)
	if overrides != nil {
		result = agent.MergeConfigs(result, overrides)
	}

	return result
}
