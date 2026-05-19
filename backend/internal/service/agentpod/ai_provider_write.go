package agentpod

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func (s *AIProviderService) CreateUserProvider(ctx context.Context, userID int64, providerType, name string, credentials map[string]string, isDefault bool) (*agentpod.UserAIProvider, error) {
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

	if isDefault {
		if err := s.repo.ClearDefaults(ctx, userID, providerType); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Create(ctx, provider); err != nil {
		slog.ErrorContext(ctx, "failed to create AI provider", "user_id", userID, "provider_type", providerType, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "AI provider created", "provider_id", provider.ID, "user_id", userID, "provider_type", providerType)
	return provider, nil
}

func (s *AIProviderService) UpdateUserProvider(ctx context.Context, providerID int64, name string, credentials map[string]string, isDefault, isEnabled bool) (*agentpod.UserAIProvider, error) {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	if len(credentials) > 0 {
		encrypted, err := s.encryptCredentials(credentials)
		if err != nil {
			return nil, err
		}
		provider.EncryptedCredentials = encrypted
	}

	provider.Name = name
	provider.IsEnabled = isEnabled

	if isDefault && !provider.IsDefault {
		if err := s.repo.ClearDefaults(ctx, provider.UserID, provider.ProviderType); err != nil {
			return nil, err
		}
		provider.IsDefault = true
	} else if !isDefault {
		provider.IsDefault = false
	}

	if err := s.repo.Save(ctx, provider); err != nil {
		slog.ErrorContext(ctx, "failed to update AI provider", "provider_id", providerID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "AI provider updated", "provider_id", providerID, "user_id", provider.UserID, "provider_type", provider.ProviderType)
	return provider, nil
}

func (s *AIProviderService) DeleteUserProvider(ctx context.Context, providerID int64) error {
	if err := s.repo.Delete(ctx, providerID); err != nil {
		slog.ErrorContext(ctx, "failed to delete AI provider", "provider_id", providerID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "AI provider deleted", "provider_id", providerID)
	return nil
}

func (s *AIProviderService) SetDefaultProvider(ctx context.Context, providerID int64) error {
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	if err := s.repo.ClearDefaults(ctx, provider.UserID, provider.ProviderType); err != nil {
		return err
	}

	if err := s.repo.SetDefault(ctx, providerID); err != nil {
		slog.ErrorContext(ctx, "failed to set default AI provider", "provider_id", providerID, "error", err)
		return err
	}
	slog.InfoContext(ctx, "AI provider set as default", "provider_id", providerID, "user_id", provider.UserID, "provider_type", provider.ProviderType)
	return nil
}
