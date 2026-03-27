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
	repo     agent.UserConfigRepository
	agentSvc AgentProvider
}

// NewUserConfigService creates a new user config service
func NewUserConfigService(repo agent.UserConfigRepository, agentSvc AgentProvider) *UserConfigService {
	return &UserConfigService{
		repo:     repo,
		agentSvc: agentSvc,
	}
}

// GetUserAgentConfig returns the user's personal config for an agent
func (s *UserConfigService) GetUserAgentConfig(ctx context.Context, userID int64, agentSlug string) (*agent.UserAgentConfig, error) {
	config, err := s.repo.GetByUserAndAgentSlug(ctx, userID, agentSlug)
	if err != nil {
		return nil, err
	}
	if config == nil {
		// Return empty config if not found
		return &agent.UserAgentConfig{
			UserID:       userID,
			AgentSlug:  agentSlug,
			ConfigValues: make(agent.ConfigValues),
		}, nil
	}
	return config, nil
}

// SetUserAgentConfig sets the user's personal config for an agent
func (s *UserConfigService) SetUserAgentConfig(ctx context.Context, userID int64, agentSlug string, configValues agent.ConfigValues) (*agent.UserAgentConfig, error) {
	// Verify agent exists
	if _, err := s.agentSvc.GetAgent(ctx, agentSlug); err != nil {
		return nil, err
	}

	if err := s.repo.Upsert(ctx, userID, agentSlug, configValues); err != nil {
		return nil, err
	}

	return s.GetUserAgentConfig(ctx, userID, agentSlug)
}

// DeleteUserAgentConfig deletes the user's personal config for an agent
func (s *UserConfigService) DeleteUserAgentConfig(ctx context.Context, userID int64, agentSlug string) error {
	return s.repo.Delete(ctx, userID, agentSlug)
}

// ListUserAgentConfigs returns all personal configs for a user
func (s *UserConfigService) ListUserAgentConfigs(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	return s.repo.ListByUser(ctx, userID)
}

// GetUserEffectiveConfig returns the effective config by merging ConfigSchema defaults and user personal config
func (s *UserConfigService) GetUserEffectiveConfig(ctx context.Context, userID int64, agentSlug string, overrides agent.ConfigValues) agent.ConfigValues {
	result := make(agent.ConfigValues)

	// 1. Get ConfigSchema defaults from Agent
	agentDef, err := s.agentSvc.GetAgent(ctx, agentSlug)
	if err == nil && agentDef.ConfigSchema.Fields != nil {
		for _, field := range agentDef.ConfigSchema.Fields {
			if field.Default != nil {
				result[field.Name] = field.Default
			}
		}
	}

	// 2. Get user's personal config
	userConfig, err := s.GetUserAgentConfig(ctx, userID, agentSlug)
	if err == nil && userConfig.ConfigValues != nil {
		result = agent.MergeConfigs(result, userConfig.ConfigValues)
	}

	// 3. Apply overrides (from CreatePod request)
	if overrides != nil {
		result = agent.MergeConfigs(result, overrides)
	}

	return result
}
