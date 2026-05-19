package agent

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

var (
	ErrUserAgentConfigNotFound = errors.New("user agent config not found")
)

type UserConfigService struct {
	repo     agent.UserConfigRepository
	agentSvc AgentProvider
}

func NewUserConfigService(repo agent.UserConfigRepository, agentSvc AgentProvider) *UserConfigService {
	return &UserConfigService{
		repo:     repo,
		agentSvc: agentSvc,
	}
}

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

func (s *UserConfigService) GetUserConfigPrefs(ctx context.Context, userID int64, agentSlug string) map[string]interface{} {
	config, err := s.GetUserAgentConfig(ctx, userID, agentSlug)
	if err != nil || config == nil {
		return nil
	}
	return map[string]interface{}(config.ConfigValues)
}

func (s *UserConfigService) SetUserAgentConfig(ctx context.Context, userID int64, agentSlug string, configValues agent.ConfigValues) (*agent.UserAgentConfig, error) {
	if _, err := s.agentSvc.GetAgent(ctx, agentSlug); err != nil {
		return nil, err
	}

	if err := s.repo.Upsert(ctx, userID, agentSlug, configValues); err != nil {
		slog.ErrorContext(ctx, "failed to set user agent config", "user_id", userID, "agent_slug", agentSlug, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "user agent config set", "user_id", userID, "agent_slug", agentSlug)
	return s.GetUserAgentConfig(ctx, userID, agentSlug)
}

func (s *UserConfigService) DeleteUserAgentConfig(ctx context.Context, userID int64, agentSlug string) error {
	if err := s.repo.Delete(ctx, userID, agentSlug); err != nil {
		slog.ErrorContext(ctx, "failed to delete user agent config", "user_id", userID, "agent_slug", agentSlug, "error", err)
		return err
	}
	slog.InfoContext(ctx, "user agent config deleted", "user_id", userID, "agent_slug", agentSlug)
	return nil
}

func (s *UserConfigService) ListUserAgentConfigs(ctx context.Context, userID int64) ([]*agent.UserAgentConfig, error) {
	return s.repo.ListByUser(ctx, userID)
}
