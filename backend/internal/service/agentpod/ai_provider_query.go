package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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

func (s *AIProviderService) GetUserProviders(ctx context.Context, userID int64) ([]*agentpod.UserAIProvider, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *AIProviderService) GetUserProvidersByType(ctx context.Context, userID int64, providerType string) ([]*agentpod.UserAIProvider, error) {
	return s.repo.ListByUserAndType(ctx, userID, providerType)
}

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
