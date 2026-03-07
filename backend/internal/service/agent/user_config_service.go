package agent

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// Errors for UserConfigService
var (
	ErrUserAgentConfigNotFound = errors.New("user agent config not found")
)

// UserConfigService handles user personal runtime configuration
type UserConfigService struct {
	repo             agent.UserConfigRepository
	agentTypeService AgentTypeProvider
}

// NewUserConfigService creates a new user config service
func NewUserConfigService(repo agent.UserConfigRepository, agentTypeService AgentTypeProvider) *UserConfigService {
	return &UserConfigService{
		repo:             repo,
		agentTypeService: agentTypeService,
	}
}

// GetUserAgentConfig returns the user's personal config for an agent type
func (s *UserConfigService) GetUserAgentConfig(ctx context.Context, userID, agentTypeID int64) (*agent.UserAgentConfig, error) {
	config, err := s.repo.GetByUserAndAgentType(ctx, userID, agentTypeID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		// Return empty config if not found
		return &agent.UserAgentConfig{
			UserID:       userID,
			AgentTypeID:  agentTypeID,
			ConfigValues: make(agent.ConfigValues),
		}, nil
	}
	return config, nil
}

// SetUserAgentConfig sets the user's personal config for an agent type
func (s *UserConfigService) SetUserAgentConfig(ctx context.Context, userID, agentTypeID int64, configValues agent.ConfigValues) (*agent.UserAgentConfig, error) {
	// Verify agent type exists
	if _, err := s.agentTypeService.GetAgentType(ctx, agentTypeID); err != nil {
		return nil, err
	}

	if err := s.repo.Upsert(ctx, userID, agentTypeID, configValues); err != nil {
		return nil, err
	}

	return s.GetUserAgentConfig(ctx, userID, agentTypeID)
}

// DeleteUserAgentConfig deletes the user's personal config for an agent type
func (s *UserConfigService) DeleteUserAgentConfig(ctx context.Context, userID, agentTypeID int64) error {
	return s.repo.Delete(ctx, userID, agentTypeID)
}

// ListUserAgentConfigs returns all personal configs for a user
func (s *UserConfigService) ListUserAgentConfigs(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	return s.repo.ListByUser(ctx, userID)
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
