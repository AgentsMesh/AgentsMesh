package agentpod

import (
	"context"
	"errors"
	"sync"

	domain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

var (
	// ErrMissingAPIKey is returned when API key is missing
	ErrMissingAPIKey = errors.New("api_key is required")
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
		UserID:             userID,
		PreparationTimeout: 300,
		TerminalFontSize:   &fontSize,
		TerminalTheme:      &theme,
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

	if updates.PreparationScript != nil {
		settings.PreparationScript = updates.PreparationScript
	}
	if updates.PreparationTimeout != nil {
		settings.PreparationTimeout = *updates.PreparationTimeout
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

// MockAIProviderService is a mock implementation of AIProviderService for testing
type MockAIProviderService struct {
	mu        sync.RWMutex
	providers map[int64]*domain.UserAIProvider
	userMap   map[int64][]int64 // userID -> provider IDs
	nextID    int64
	err       error
}

// NewMockAIProviderService creates a new mock AI provider service
func NewMockAIProviderService() *MockAIProviderService {
	return &MockAIProviderService{
		providers: make(map[int64]*domain.UserAIProvider),
		userMap:   make(map[int64][]int64),
		nextID:    1,
	}
}

// SetError sets the error to return from all operations
func (m *MockAIProviderService) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// AddProvider adds a provider for testing
func (m *MockAIProviderService) AddProvider(provider *domain.UserAIProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if provider.ID == 0 {
		provider.ID = m.nextID
		m.nextID++
	}
	m.providers[provider.ID] = provider
	m.userMap[provider.UserID] = append(m.userMap[provider.UserID], provider.ID)
}

// GetUserProviders implements AIProviderService
func (m *MockAIProviderService) GetUserProviders(ctx context.Context, userID int64) ([]*domain.UserAIProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	var result []*domain.UserAIProvider
	if ids, ok := m.userMap[userID]; ok {
		for _, id := range ids {
			if p, ok := m.providers[id]; ok {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

// CreateUserProvider implements AIProviderService
func (m *MockAIProviderService) CreateUserProvider(
	ctx context.Context,
	userID int64,
	providerType string,
	name string,
	credentials map[string]string,
	isDefault bool,
) (*domain.UserAIProvider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	provider := &domain.UserAIProvider{
		ID:           m.nextID,
		UserID:       userID,
		ProviderType: providerType,
		Name:         name,
		IsDefault:    isDefault,
		IsEnabled:    true,
	}
	m.nextID++
	m.providers[provider.ID] = provider
	m.userMap[userID] = append(m.userMap[userID], provider.ID)

	return provider, nil
}

// UpdateUserProvider implements AIProviderService
func (m *MockAIProviderService) UpdateUserProvider(
	ctx context.Context,
	providerID int64,
	name string,
	credentials map[string]string,
	isDefault bool,
	isEnabled bool,
) (*domain.UserAIProvider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	provider, ok := m.providers[providerID]
	if !ok {
		return nil, ErrProviderNotFound
	}

	if name != "" {
		provider.Name = name
	}
	provider.IsDefault = isDefault
	provider.IsEnabled = isEnabled

	return provider, nil
}

// DeleteUserProvider implements AIProviderService
func (m *MockAIProviderService) DeleteUserProvider(ctx context.Context, providerID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}

	if _, ok := m.providers[providerID]; !ok {
		return ErrProviderNotFound
	}

	delete(m.providers, providerID)
	return nil
}

// SetDefaultProvider implements AIProviderService
func (m *MockAIProviderService) SetDefaultProvider(ctx context.Context, providerID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}

	provider, ok := m.providers[providerID]
	if !ok {
		return ErrProviderNotFound
	}

	// Unset other defaults of the same type
	for _, p := range m.providers {
		if p.UserID == provider.UserID && p.ProviderType == provider.ProviderType {
			p.IsDefault = false
		}
	}
	provider.IsDefault = true

	return nil
}

// ValidateCredentials validates provider credentials
func (m *MockAIProviderService) ValidateCredentials(providerType string, credentials map[string]string) error {
	if m.err != nil {
		return m.err
	}

	switch providerType {
	case domain.AIProviderTypeClaude:
		if _, ok := credentials["api_key"]; !ok {
			return ErrMissingAPIKey
		}
	case domain.AIProviderTypeGemini:
		if _, ok := credentials["api_key"]; !ok {
			return ErrMissingAPIKey
		}
	case domain.AIProviderTypeOpenAI:
		if _, ok := credentials["api_key"]; !ok {
			return ErrMissingAPIKey
		}
	}
	return nil
}

// GetAIProviderEnvVars implements AIProviderService
func (m *MockAIProviderService) GetAIProviderEnvVars(ctx context.Context, userID int64) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	// Find default provider
	for _, id := range m.userMap[userID] {
		p := m.providers[id]
		if p.IsDefault && p.IsEnabled {
			return map[string]string{
				"ANTHROPIC_API_KEY": "test-key",
			}, nil
		}
	}

	return nil, nil
}
