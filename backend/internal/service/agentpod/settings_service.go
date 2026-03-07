package agentpod

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

var (
	ErrSettingsNotFound = errors.New("user AgentPod settings not found")
)

// SettingsService handles user AgentPod settings operations
type SettingsService struct {
	repo agentpod.SettingsRepository
}

// NewSettingsService creates a new settings service
func NewSettingsService(repo agentpod.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// GetUserSettings returns the AgentPod settings for a user
// Creates default settings if none exist
func (s *SettingsService) GetUserSettings(ctx context.Context, userID int64) (*agentpod.UserAgentPodSettings, error) {
	settings, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		// Create default settings
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

// UpdateUserSettings updates the AgentPod settings for a user
func (s *SettingsService) UpdateUserSettings(ctx context.Context, userID int64, updates *UserSettingsUpdate) (*agentpod.UserAgentPodSettings, error) {
	settings, err := s.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if updates.DefaultAgentTypeID != nil {
		settings.DefaultAgentTypeID = updates.DefaultAgentTypeID
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
		return nil, err
	}

	return settings, nil
}

// DeleteUserSettings removes AgentPod settings for a user
func (s *SettingsService) DeleteUserSettings(ctx context.Context, userID int64) error {
	return s.repo.DeleteByUserID(ctx, userID)
}

// UserSettingsUpdate represents partial updates to user settings
type UserSettingsUpdate struct {
	DefaultAgentTypeID *int64  `json:"default_agent_type_id,omitempty"`
	DefaultModel       *string `json:"default_model,omitempty"`
	DefaultPermMode    *string `json:"default_perm_mode,omitempty"`
	TerminalFontSize   *int    `json:"terminal_font_size,omitempty"`
	TerminalTheme      *string `json:"terminal_theme,omitempty"`
}


// GetDefaultAgentConfig returns the default agent configuration for a user
func (s *SettingsService) GetDefaultAgentConfig(ctx context.Context, userID int64) (*DefaultAgentConfig, error) {
	settings, err := s.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &DefaultAgentConfig{
		AgentTypeID: settings.DefaultAgentTypeID,
		Model:       settings.DefaultModel,
		PermMode:    settings.DefaultPermMode,
	}, nil
}

// DefaultAgentConfig represents default agent settings
type DefaultAgentConfig struct {
	AgentTypeID *int64  `json:"agent_type_id,omitempty"`
	Model       *string `json:"model,omitempty"`
	PermMode    *string `json:"perm_mode,omitempty"`
}

// GetTerminalPreferences returns terminal UI preferences for a user
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

// TerminalPreferences represents terminal UI settings
type TerminalPreferences struct {
	FontSize *int    `json:"font_size,omitempty"`
	Theme    *string `json:"theme,omitempty"`
}
