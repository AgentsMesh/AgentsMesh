package agentpod

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

// GetUserDefaultCredentials returns the default credentials for a user and provider type
func (s *AIProviderService) GetUserDefaultCredentials(ctx context.Context, userID int64, providerType string) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ? AND is_default = ? AND is_enabled = ?",
			userID, providerType, true, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}

// GetAIProviderEnvVars returns AI provider credentials as environment variables for a user
// This retrieves the user's default provider credentials and formats them for PTY injection
func (s *AIProviderService) GetAIProviderEnvVars(ctx context.Context, userID int64) (map[string]string, error) {
	// Try to get default Claude credentials first
	credentials, err := s.GetUserDefaultCredentials(ctx, userID, agentpod.AIProviderTypeClaude)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return nil, nil // No credentials configured
		}
		return nil, err
	}

	return s.formatEnvVars(agentpod.AIProviderTypeClaude, credentials), nil
}

// GetAIProviderEnvVarsByID returns AI provider credentials as environment variables by provider ID
func (s *AIProviderService) GetAIProviderEnvVarsByID(ctx context.Context, providerID int64) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("id = ? AND is_enabled = ?", providerID, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	credentials, err := s.decryptCredentials(provider.EncryptedCredentials)
	if err != nil {
		return nil, err
	}

	return s.formatEnvVars(provider.ProviderType, credentials), nil
}

// GetUserProviders returns all AI providers for a user
func (s *AIProviderService) GetUserProviders(ctx context.Context, userID int64) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("provider_type, name").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetUserProvidersByType returns AI providers for a user filtered by type
func (s *AIProviderService) GetUserProvidersByType(ctx context.Context, userID int64, providerType string) ([]*agentpod.UserAIProvider, error) {
	var providers []*agentpod.UserAIProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND provider_type = ?", userID, providerType).
		Order("is_default DESC, name").
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetProviderCredentials returns decrypted credentials for a provider
// This should only be used when the credentials need to be displayed/edited
func (s *AIProviderService) GetProviderCredentials(ctx context.Context, providerID int64) (map[string]string, error) {
	var provider agentpod.UserAIProvider
	if err := s.db.WithContext(ctx).First(&provider, providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}
