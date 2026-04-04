package agent

import (
	"context"
	"errors"
	"log/slog"

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
		return &agent.UserAgentConfig{
			UserID:       userID,
			AgentSlug:  agentSlug,
			ConfigValues: make(agent.ConfigValues),
		}, nil
	}
	return config, nil
}

// GetUserConfigPrefs returns the user's personal config preferences as a plain map.
// Implements UserConfigQueryForOrchestrator for AgentFile resolve integration.
func (s *UserConfigService) GetUserConfigPrefs(ctx context.Context, userID int64, agentSlug string) map[string]interface{} {
	config, err := s.GetUserAgentConfig(ctx, userID, agentSlug)
	if err != nil || config == nil {
		return nil
	}
	return map[string]interface{}(config.ConfigValues)
}

// SetUserAgentConfig sets the user's personal config for an agent
func (s *UserConfigService) SetUserAgentConfig(ctx context.Context, userID int64, agentSlug string, configValues agent.ConfigValues) (*agent.UserAgentConfig, error) {
	if _, err := s.agentSvc.GetAgent(ctx, agentSlug); err != nil {
		return nil, err
	}

	if err := s.repo.Upsert(ctx, userID, agentSlug, configValues); err != nil {
		slog.Error("failed to set user agent config", "user_id", userID, "agent_slug", agentSlug, "error", err)
		return nil, err
	}

	slog.Info("user agent config set", "user_id", userID, "agent_slug", agentSlug)
	return s.GetUserAgentConfig(ctx, userID, agentSlug)
}

// DeleteUserAgentConfig deletes the user's personal config for an agent
func (s *UserConfigService) DeleteUserAgentConfig(ctx context.Context, userID int64, agentSlug string) error {
	if err := s.repo.Delete(ctx, userID, agentSlug); err != nil {
		slog.Error("failed to delete user agent config", "user_id", userID, "agent_slug", agentSlug, "error", err)
		return err
	}
	slog.Info("user agent config deleted", "user_id", userID, "agent_slug", agentSlug)
	return nil
}

// ListUserAgentConfigs returns all personal configs for a user
func (s *UserConfigService) ListUserAgentConfigs(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	return s.repo.ListByUser(ctx, userID)
}
