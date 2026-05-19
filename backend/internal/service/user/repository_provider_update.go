package user

import (
	"context"
	"log/slog"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

type UpdateRepositoryProviderRequest struct {
	Name         *string
	BaseURL      *string
	ClientID     *string
	ClientSecret *string // Plain text, will be encrypted
	BotToken     *string // Plain text, will be encrypted
	IsActive     *bool
}

func (s *Service) UpdateRepositoryProvider(ctx context.Context, userID, providerID int64, req *UpdateRepositoryProviderRequest) (*domainUser.RepositoryProvider, error) {
	provider, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != "" {
		exists, err := s.repo.RepositoryProviderNameExists(ctx, userID, *req.Name, &providerID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrProviderAlreadyExists
		}
		updates["name"] = *req.Name
	}

	if req.BaseURL != nil {
		updates["base_url"] = *req.BaseURL
	}

	if req.ClientID != nil {
		if *req.ClientID == "" {
			updates["client_id"] = nil
		} else {
			updates["client_id"] = *req.ClientID
		}
	}

	if req.ClientSecret != nil {
		if *req.ClientSecret == "" {
			updates["client_secret_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.ClientSecret, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt client secret", "user_id", userID, "provider_id", providerID, "error", err)
				return nil, err
			}
			updates["client_secret_encrypted"] = encrypted
		} else {
			updates["client_secret_encrypted"] = *req.ClientSecret
		}
	}

	if req.BotToken != nil {
		if *req.BotToken == "" {
			updates["bot_token_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.BotToken, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt bot token", "user_id", userID, "provider_id", providerID, "error", err)
				return nil, err
			}
			updates["bot_token_encrypted"] = encrypted
		} else {
			updates["bot_token_encrypted"] = *req.BotToken
		}
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		return provider, nil
	}

	if err := s.repo.UpdateRepositoryProvider(ctx, provider, updates); err != nil {
		slog.ErrorContext(ctx, "failed to update repository provider", "user_id", userID, "provider_id", providerID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "repository provider updated", "user_id", userID, "provider_id", providerID)

	return s.GetRepositoryProvider(ctx, userID, providerID)
}

func (s *Service) SetDefaultRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	_, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return err
	}

	return s.repo.SetDefaultRepositoryProvider(ctx, userID, providerID)
}
