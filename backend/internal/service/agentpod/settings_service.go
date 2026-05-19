package agentpod

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

var (
	ErrSettingsNotFound = errors.New("user AgentPod settings not found")
)

type SettingsService struct {
	repo agentpod.SettingsRepository
}

func NewSettingsService(repo agentpod.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) GetUserSettings(ctx context.Context, userID int64) (*agentpod.UserAgentPodSettings, error) {
	settings, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		settings = &agentpod.UserAgentPodSettings{
			UserID: userID,
		}
		if err := s.repo.Create(ctx, settings); err != nil {
			return nil, err
		}
		return settings, nil
	}

	return settings, nil
}

func (s *SettingsService) UpdateUserSettings(ctx context.Context, userID int64, updates *UserSettingsUpdate) (*agentpod.UserAgentPodSettings, error) {
	settings, err := s.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if updates.DefaultAgentSlug != nil {
		settings.DefaultAgentSlug = updates.DefaultAgentSlug
	}
	if updates.DefaultModel != nil {
		settings.DefaultModel = updates.DefaultModel
	}
	if updates.DefaultPermMode != nil {
		settings.DefaultPermMode = updates.DefaultPermMode
	}
	if updates.TerminalFontSize != nil {
		settings.TerminalFontSize = updates.TerminalFontSize
	}
	if updates.TerminalTheme != nil {
		settings.TerminalTheme = updates.TerminalTheme
	}

	if err := s.repo.Save(ctx, settings); err != nil {
		slog.ErrorContext(ctx, "failed to update user settings", "user_id", userID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "user settings updated", "user_id", userID)
	return settings, nil
}

func (s *SettingsService) DeleteUserSettings(ctx context.Context, userID int64) error {
	return s.repo.DeleteByUserID(ctx, userID)
}

type UserSettingsUpdate struct {
	DefaultAgentSlug *string `json:"default_agent_slug,omitempty"`
	DefaultModel       *string `json:"default_model,omitempty"`
	DefaultPermMode    *string `json:"default_perm_mode,omitempty"`
	TerminalFontSize   *int    `json:"terminal_font_size,omitempty"`
	TerminalTheme      *string `json:"terminal_theme,omitempty"`
}

func (s *SettingsService) GetDefaultAgentConfig(ctx context.Context, userID int64) (*DefaultAgentConfig, error) {
	settings, err := s.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &DefaultAgentConfig{
		AgentSlug: settings.DefaultAgentSlug,
		Model:       settings.DefaultModel,
		PermMode:    settings.DefaultPermMode,
	}, nil
}

type DefaultAgentConfig struct {
	AgentSlug *string `json:"agent_slug,omitempty"`
	Model       *string `json:"model,omitempty"`
	PermMode    *string `json:"perm_mode,omitempty"`
}

func (s *SettingsService) GetTerminalPreferences(ctx context.Context, userID int64) (*TerminalPreferences, error) {
	settings, err := s.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &TerminalPreferences{
		FontSize: settings.TerminalFontSize,
		Theme:    settings.TerminalTheme,
	}, nil
}

type TerminalPreferences struct {
	FontSize *int    `json:"font_size,omitempty"`
	Theme    *string `json:"theme,omitempty"`
}
