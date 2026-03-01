package agentpod

import (
	"context"
	"sync"

	domain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
)

// MockSettingsService is a mock implementation of SettingsService for testing
type MockSettingsService struct {
	mu       sync.RWMutex
	settings map[int64]*domain.UserAgentPodSettings
	err      error
}

// NewMockSettingsService creates a new mock settings service
func NewMockSettingsService() *MockSettingsService {
	return &MockSettingsService{
		settings: make(map[int64]*domain.UserAgentPodSettings),
	}
}

// SetError sets the error to return from all operations
func (m *MockSettingsService) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// AddSettings adds settings for testing
func (m *MockSettingsService) AddSettings(settings *domain.UserAgentPodSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[settings.UserID] = settings
}

// GetUserSettings implements SettingsService
func (m *MockSettingsService) GetUserSettings(ctx context.Context, userID int64) (*domain.UserAgentPodSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	if settings, ok := m.settings[userID]; ok {
		return settings, nil
	}

	// Return default settings
	fontSize := 14
	theme := "dark"
	return &domain.UserAgentPodSettings{
		UserID:           userID,
		TerminalFontSize: &fontSize,
		TerminalTheme:    &theme,
	}, nil
}

// UpdateUserSettings implements SettingsService
func (m *MockSettingsService) UpdateUserSettings(ctx context.Context, userID int64, updates *UserSettingsUpdate) (*domain.UserAgentPodSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	settings, ok := m.settings[userID]
	if !ok {
		settings = &domain.UserAgentPodSettings{UserID: userID}
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

	m.settings[userID] = settings
	return settings, nil
}
