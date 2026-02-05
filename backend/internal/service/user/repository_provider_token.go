package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

// GetDecryptedProviderToken retrieves and decrypts the access token for a repository provider
// It first checks if the provider has a linked OAuth identity, then falls back to bot token
func (s *Service) GetDecryptedProviderToken(ctx context.Context, userID, providerID int64) (string, error) {
	// Get provider with Identity preloaded
	var provider user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Preload("Identity").
		Where("id = ? AND user_id = ?", providerID, userID).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrProviderNotFound
		}
		return "", err
	}

	// 1. Try OAuth identity token first
	if provider.IdentityID != nil && provider.Identity != nil {
		if provider.Identity.AccessTokenEncrypted != nil && *provider.Identity.AccessTokenEncrypted != "" {
			if s.encryptionKey != "" {
				return crypto.DecryptWithKey(*provider.Identity.AccessTokenEncrypted, s.encryptionKey)
			}
			return *provider.Identity.AccessTokenEncrypted, nil
		}
	}

	// 2. Fall back to bot token
	if provider.BotTokenEncrypted != nil && *provider.BotTokenEncrypted != "" {
		if s.encryptionKey != "" {
			return crypto.DecryptWithKey(*provider.BotTokenEncrypted, s.encryptionKey)
		}
		return *provider.BotTokenEncrypted, nil
	}

	return "", nil
}

// GetRepositoryProviderByTypeAndURL returns a repository provider by provider type and base URL
func (s *Service) GetRepositoryProviderByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Preload("Identity").
		Where("user_id = ? AND provider_type = ? AND base_url = ? AND is_active = ?", userID, providerType, baseURL, true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// GetDecryptedProviderTokenByTypeAndURL retrieves the access token for a repository provider
// It first checks if the provider has a linked OAuth identity, then falls back to bot token
func (s *Service) GetDecryptedProviderTokenByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (string, error) {
	provider, err := s.GetRepositoryProviderByTypeAndURL(ctx, userID, providerType, baseURL)
	if err != nil {
		return "", err
	}

	// 1. Try OAuth identity token first
	if provider.IdentityID != nil && provider.Identity != nil {
		if provider.Identity.AccessTokenEncrypted != nil && *provider.Identity.AccessTokenEncrypted != "" {
			if s.encryptionKey != "" {
				return crypto.DecryptWithKey(*provider.Identity.AccessTokenEncrypted, s.encryptionKey)
			}
			return *provider.Identity.AccessTokenEncrypted, nil
		}
	}

	// 2. Fall back to bot token
	if provider.BotTokenEncrypted != nil && *provider.BotTokenEncrypted != "" {
		if s.encryptionKey != "" {
			return crypto.DecryptWithKey(*provider.BotTokenEncrypted, s.encryptionKey)
		}
		return *provider.BotTokenEncrypted, nil
	}

	return "", nil
}

// EnsureRepositoryProviderForIdentity ensures a RepositoryProvider exists for an OAuth identity
// This is called during OAuth login to automatically create a provider linked to the identity
func (s *Service) EnsureRepositoryProviderForIdentity(ctx context.Context, userID int64, provider string) error {
	// 1. Get user's identity for this provider
	identity, err := s.GetIdentityByProvider(ctx, userID, provider)
	if err != nil {
		return err
	}

	// 2. Check if a provider already exists linked to this identity
	var existing user.RepositoryProvider
	err = s.db.WithContext(ctx).
		Where("user_id = ? AND identity_id = ?", userID, identity.ID).
		First(&existing).Error
	if err == nil {
		// Provider already exists, nothing to do
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 3. Create new provider linked to identity
	baseURL := getDefaultBaseURL(provider)
	name := getDefaultProviderName(provider)

	// 4. Ensure unique name - if name already exists, append a suffix
	name = s.ensureUniqueProviderName(ctx, userID, name)

	newProvider := &user.RepositoryProvider{
		UserID:       userID,
		ProviderType: provider,
		Name:         name,
		BaseURL:      baseURL,
		IdentityID:   &identity.ID,
		IsActive:     true,
	}

	return s.db.WithContext(ctx).Create(newProvider).Error
}

// ensureUniqueProviderName returns a unique provider name for the user
// If the name already exists, it appends a numeric suffix (e.g., "GitHub (2)")
func (s *Service) ensureUniqueProviderName(ctx context.Context, userID int64, baseName string) string {
	var existing user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND name = ?", userID, baseName).
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return baseName // Name is available
	}

	// Name exists, find a unique suffix
	for i := 2; i <= 100; i++ {
		candidateName := fmt.Sprintf("%s (%d)", baseName, i)
		err := s.db.WithContext(ctx).
			Where("user_id = ? AND name = ?", userID, candidateName).
			First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidateName
		}
	}

	// Fallback: use timestamp (extremely unlikely to reach here)
	return baseName + " (OAuth)"
}

// getDefaultBaseURL returns the default base URL for a provider type
func getDefaultBaseURL(provider string) string {
	switch provider {
	case user.ProviderTypeGitHub:
		return "https://github.com"
	case user.ProviderTypeGitLab:
		return "https://gitlab.com"
	case user.ProviderTypeGitee:
		return "https://gitee.com"
	default:
		return ""
	}
}

// getDefaultProviderName returns the default display name for a provider type
func getDefaultProviderName(provider string) string {
	switch provider {
	case user.ProviderTypeGitHub:
		return "GitHub"
	case user.ProviderTypeGitLab:
		return "GitLab"
	case user.ProviderTypeGitee:
		return "Gitee"
	default:
		return provider
	}
}
