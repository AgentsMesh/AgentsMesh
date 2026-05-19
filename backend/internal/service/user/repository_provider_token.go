package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

func (s *Service) GetDecryptedProviderToken(ctx context.Context, userID, providerID int64) (string, error) {
	provider, err := s.repo.GetRepositoryProviderWithIdentity(ctx, userID, providerID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return "", ErrProviderNotFound
		}
		return "", err
	}

	return s.decryptProviderToken(provider)
}

func (s *Service) GetRepositoryProviderByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (*user.RepositoryProvider, error) {
	provider, err := s.repo.GetRepositoryProviderByTypeAndURL(ctx, userID, providerType, baseURL)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return provider, nil
}

func (s *Service) GetDecryptedProviderTokenByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (string, error) {
	provider, err := s.GetRepositoryProviderByTypeAndURL(ctx, userID, providerType, baseURL)
	if err != nil {
		return "", err
	}

	return s.decryptProviderToken(provider)
}

func (s *Service) decryptProviderToken(provider *user.RepositoryProvider) (string, error) {
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

func (s *Service) EnsureRepositoryProviderForIdentity(ctx context.Context, userID int64, provider string) error {
	identity, err := s.GetIdentityByProvider(ctx, userID, provider)
	if err != nil {
		return err
	}

	_, err = s.repo.GetRepositoryProviderByIdentityID(ctx, userID, identity.ID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, user.ErrNotFound) {
		return err
	}

	baseURL := getDefaultBaseURL(provider)
	name := getDefaultProviderName(provider)

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
		slog.ErrorContext(ctx, "failed to create repository provider for identity",
			"user_id", userID, "provider", provider, "error", err)
		return err
	}
	slog.InfoContext(ctx, "repository provider created for oauth identity",
		"user_id", userID, "provider", provider, "provider_id", newProvider.ID)
	return nil
}

func (s *Service) ensureUniqueProviderName(ctx context.Context, userID int64, baseName string) string {
	exists, _ := s.repo.RepositoryProviderNameExists(ctx, userID, baseName, nil)
	if !exists {
		return baseName // Name is available
	}

	for i := 2; i <= 100; i++ {
		candidateName := fmt.Sprintf("%s (%d)", baseName, i)
		exists, _ := s.repo.RepositoryProviderNameExists(ctx, userID, candidateName, nil)
		if !exists {
			return candidateName
		}
	}

	return baseName + " (OAuth)"
}

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
