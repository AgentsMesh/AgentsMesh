package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// GetUserDefaultCredentials returns the default credentials for a user and provider type
func (s *AIProviderService) GetUserDefaultCredentials(ctx context.Context, userID int64, providerType string) (map[string]string, error) {
	provider, err := s.repo.GetDefaultByType(ctx, userID, providerType)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}

// GetAIProviderEnvVars returns AI provider credentials as environment variables for a user
func (s *AIProviderService) GetAIProviderEnvVars(ctx context.Context, userID int64) (map[string]string, error) {
	credentials, err := s.GetUserDefaultCredentials(ctx, userID, agentpod.AIProviderTypeClaude)
	if err != nil {
		if err == ErrProviderNotFound {
			return nil, nil // No credentials configured
		}
		return nil, err
	}

	return s.formatEnvVars(agentpod.AIProviderTypeClaude, credentials), nil
}

// GetAIProviderEnvVarsByID returns AI provider credentials as environment variables by provider ID
func (s *AIProviderService) GetAIProviderEnvVarsByID(ctx context.Context, providerID int64) (map[string]string, error) {
	provider, err := s.repo.GetEnabledByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	credentials, err := s.decryptCredentials(provider.EncryptedCredentials)
	if err != nil {
		return nil, err
	}

	return s.formatEnvVars(provider.ProviderType, credentials), nil
}

// GetUserProviders returns all AI providers for a user
func (s *AIProviderService) GetUserProviders(ctx context.Context, userID int64) ([]*agentpod.UserAIProvider, error) {
	return s.repo.ListByUser(ctx, userID)
}

// GetUserProvidersByType returns AI providers for a user filtered by type
func (s *AIProviderService) GetUserProvidersByType(ctx context.Context, userID int64, providerType string) ([]*agentpod.UserAIProvider, error) {
	return s.repo.ListByUserAndType(ctx, userID, providerType)
}

// GetProviderCredentials returns decrypted credentials for a provider
func (s *AIProviderService) GetProviderCredentials(ctx context.Context, providerID int64) (map[string]string, error) {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	return s.decryptCredentials(provider.EncryptedCredentials)
}
