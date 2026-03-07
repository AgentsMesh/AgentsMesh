package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// CreateUserProvider creates a new AI provider for a user
func (s *AIProviderService) CreateUserProvider(ctx context.Context, userID int64, providerType, name string, credentials map[string]string, isDefault bool) (*agentpod.UserAIProvider, error) {
	// Encrypt credentials
	encrypted, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, err
	}

	provider := &agentpod.UserAIProvider{
		UserID:               userID,
		ProviderType:         providerType,
		Name:                 name,
		IsDefault:            isDefault,
		IsEnabled:            true,
		EncryptedCredentials: encrypted,
	}

	// If this is set as default, clear other defaults for this provider type
	if isDefault {
		if err := s.repo.ClearDefaults(ctx, userID, providerType); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// UpdateUserProvider updates an existing AI provider
func (s *AIProviderService) UpdateUserProvider(ctx context.Context, providerID int64, name string, credentials map[string]string, isDefault, isEnabled bool) (*agentpod.UserAIProvider, error) {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	// Encrypt new credentials if provided
	if len(credentials) > 0 {
		encrypted, err := s.encryptCredentials(credentials)
		if err != nil {
			return nil, err
		}
		provider.EncryptedCredentials = encrypted
	}

	provider.Name = name
	provider.IsEnabled = isEnabled

	// Handle default flag
	if isDefault && !provider.IsDefault {
		if err := s.repo.ClearDefaults(ctx, provider.UserID, provider.ProviderType); err != nil {
			return nil, err
		}
		provider.IsDefault = true
	} else if !isDefault {
		provider.IsDefault = false
	}

	if err := s.repo.Save(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// DeleteUserProvider deletes an AI provider
func (s *AIProviderService) DeleteUserProvider(ctx context.Context, providerID int64) error {
	return s.repo.Delete(ctx, providerID)
}

// SetDefaultProvider sets a provider as the default for its type
func (s *AIProviderService) SetDefaultProvider(ctx context.Context, providerID int64) error {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	// Clear other defaults
	if err := s.repo.ClearDefaults(ctx, provider.UserID, provider.ProviderType); err != nil {
		return err
	}

	// Set this one as default
	return s.repo.SetDefault(ctx, providerID)
}
