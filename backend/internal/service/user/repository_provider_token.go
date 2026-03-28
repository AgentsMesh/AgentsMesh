package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// GetDecryptedProviderToken retrieves and decrypts the access token for a repository provider
// It first checks if the provider has a linked OAuth identity, then falls back to bot token
func (s *Service) GetDecryptedProviderToken(ctx context.Context, userID, providerID int64) (string, error) {
	// Get provider with Identity preloaded
	provider, err := s.repo.GetRepositoryProviderWithIdentity(ctx, userID, providerID)
	if err != nil {
		if err == user.ErrNotFound {
			return "", ErrProviderNotFound
		}
		return "", err
	}

	return s.decryptProviderToken(provider)
}

// GetRepositoryProviderByTypeAndURL returns a repository provider by provider type and base URL
func (s *Service) GetRepositoryProviderByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (*user.RepositoryProvider, error) {
	provider, err := s.repo.GetRepositoryProviderByTypeAndURL(ctx, userID, providerType, baseURL)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return provider, nil
}

// GetDecryptedProviderTokenByTypeAndURL retrieves the access token for a repository provider
// It first checks if the provider has a linked OAuth identity, then falls back to bot token
func (s *Service) GetDecryptedProviderTokenByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (string, error) {
	provider, err := s.GetRepositoryProviderByTypeAndURL(ctx, userID, providerType, baseURL)
	if err != nil {
		return "", err
	}

	return s.decryptProviderToken(provider)
}

// decryptProviderToken extracts and decrypts the token from a provider
func (s *Service) decryptProviderToken(provider *user.RepositoryProvider) (string, error) {
	// 1. Try OAuth identity token first
	if provider.IdentityID != nil && provider.Identity != nil {
		if provider.Identity.AccessTokenEncrypted != nil && *provider.Identity.AccessTokenEncrypted != "" {
			if s.encryptionKey != "" {
				decrypted, err := crypto.DecryptWithKey(*provider.Identity.AccessTokenEncrypted, s.encryptionKey)
				if err != nil {
					slog.Error("failed to decrypt identity access token",
						"provider_id", provider.ID, "identity_id", *provider.IdentityID, "error", err)
					return "", err
				}
				return decrypted, nil
			}
			return *provider.Identity.AccessTokenEncrypted, nil
		}
	}

	// 2. Fall back to bot token
	if provider.BotTokenEncrypted != nil && *provider.BotTokenEncrypted != "" {
		if s.encryptionKey != "" {
			decrypted, err := crypto.DecryptWithKey(*provider.BotTokenEncrypted, s.encryptionKey)
			if err != nil {
				slog.Error("failed to decrypt bot token",
					"provider_id", provider.ID, "error", err)
				return "", err
			}
			return decrypted, nil
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
	_, err = s.repo.GetRepositoryProviderByIdentityID(ctx, userID, identity.ID)
	if err == nil {
		// Provider already exists, nothing to do
		return nil
	}
	if !errors.Is(err, user.ErrNotFound) {
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

	if err := s.repo.CreateRepositoryProvider(ctx, newProvider); err != nil {
		slog.Error("failed to create repository provider for identity",
			"user_id", userID, "provider", provider, "error", err)
		return err
	}
	slog.Info("repository provider created for oauth identity",
		"user_id", userID, "provider", provider, "provider_id", newProvider.ID)
	return nil
}

// ensureUniqueProviderName returns a unique provider name for the user
// If the name already exists, it appends a numeric suffix (e.g., "GitHub (2)")
func (s *Service) ensureUniqueProviderName(ctx context.Context, userID int64, baseName string) string {
	exists, _ := s.repo.RepositoryProviderNameExists(ctx, userID, baseName, nil)
	if !exists {
		return baseName // Name is available
	}

	// Name exists, find a unique suffix
	for i := 2; i <= 100; i++ {
		candidateName := fmt.Sprintf("%s (%d)", baseName, i)
		exists, _ := s.repo.RepositoryProviderNameExists(ctx, userID, candidateName, nil)
		if !exists {
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
