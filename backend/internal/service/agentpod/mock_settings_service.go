package agentpod

import (
	"context"
	"sync"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

type MockSettingsService struct {
	mu       sync.RWMutex
	settings map[int64]*domain.UserAgentPodSettings
	err      error
}

func NewMockSettingsService() *MockSettingsService {
	return &MockSettingsService{
		settings: make(map[int64]*domain.UserAgentPodSettings),
	}
}

func (m *MockSettingsService) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MockSettingsService) AddSettings(settings *domain.UserAgentPodSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[settings.UserID] = settings
}

func (m *MockSettingsService) GetUserSettings(ctx context.Context, userID int64) (*domain.UserAgentPodSettings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	if settings, ok := m.settings[userID]; ok {
		return settings, nil
	}

	fontSize := 14
	theme := "dark"
	return &domain.UserAgentPodSettings{
		UserID:           userID,
		TerminalFontSize: &fontSize,
		TerminalTheme:    &theme,
	}, nil
}

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
