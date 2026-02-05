package agentpod

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
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
		if err := s.clearDefaultProvider(ctx, userID, providerType); err != nil {
			return nil, err
		}
	}

	if err := s.db.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// UpdateUserProvider updates an existing AI provider
func (s *AIProviderService) UpdateUserProvider(ctx context.Context, providerID int64, name string, credentials map[string]string, isDefault, isEnabled bool) (*agentpod.UserAIProvider, error) {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
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
		if err := s.clearDefaultProvider(ctx, provider.UserID, provider.ProviderType); err != nil {
			return nil, err
		}
		provider.IsDefault = true
	} else if !isDefault {
		provider.IsDefault = false
	}

	if err := s.db.WithContext(ctx).Save(&provider).Error; err != nil {
		return nil, err
	}

	return &provider, nil
}

// DeleteUserProvider deletes an AI provider
func (s *AIProviderService) DeleteUserProvider(ctx context.Context, providerID int64) error {
	return s.db.WithContext(ctx).Delete(&agentpod.UserAIProvider{}, providerID).Error
}

// SetDefaultProvider sets a provider as the default for its type
func (s *AIProviderService) SetDefaultProvider(ctx context.Context, providerID int64) error {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProviderNotFound
		}
		return err
	}

	// Clear other defaults
	if err := s.clearDefaultProvider(ctx, provider.UserID, provider.ProviderType); err != nil {
		return err
	}

	// Set this one as default
	return s.db.WithContext(ctx).Model(&provider).Update("is_default", true).Error
}

// clearDefaultProvider clears the default flag for all providers of a type
func (s *AIProviderService) clearDefaultProvider(ctx context.Context, userID int64, providerType string) error {
	return s.db.WithContext(ctx).
		Model(&agentpod.UserAIProvider{}).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Update("is_default", false).Error
}
