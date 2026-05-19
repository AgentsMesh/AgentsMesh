package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

var (
	ErrProviderNotFound      = errors.New("repository provider not found")
	ErrProviderAlreadyExists = errors.New("repository provider already exists with this name")
	ErrInvalidProviderType   = errors.New("invalid provider type")
)

type CreateRepositoryProviderRequest struct {
	ProviderType string
	Name         string
	BaseURL      string
	ClientID     string
	ClientSecret string // Plain text, will be encrypted
	BotToken     string // Plain text, will be encrypted
}

func (s *Service) CreateRepositoryProvider(ctx context.Context, userID int64, req *CreateRepositoryProviderRequest) (*user.RepositoryProvider, error) {
	if !user.IsValidProviderType(req.ProviderType) {
		return nil, ErrInvalidProviderType
	}

	exists, err := s.repo.RepositoryProviderNameExists(ctx, userID, req.Name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProviderAlreadyExists
	}

	provider := &user.RepositoryProvider{
		UserID:       userID,
		ProviderType: req.ProviderType,
		Name:         req.Name,
		BaseURL:      req.BaseURL,
		IsDefault:    false,
		IsActive:     true,
	}

	if req.ClientID != "" {
		provider.ClientID = &req.ClientID
	}

	if s.encryptionKey != "" {
		if req.ClientSecret != "" {
			encrypted, err := crypto.EncryptWithKey(req.ClientSecret, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt client secret",
					"user_id", userID, "provider_type", req.ProviderType, "error", err)
				return nil, err
			}
			provider.ClientSecretEncrypted = &encrypted
		}
		if req.BotToken != "" {
			encrypted, err := crypto.EncryptWithKey(req.BotToken, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt bot token",
					"user_id", userID, "provider_type", req.ProviderType, "error", err)
				return nil, err
			}
			provider.BotTokenEncrypted = &encrypted
		}
	} else {
		slog.WarnContext(ctx, "storing provider secrets without encryption",
			"user_id", userID, "provider_type", req.ProviderType)
		if req.ClientSecret != "" {
			provider.ClientSecretEncrypted = &req.ClientSecret
		}
		if req.BotToken != "" {
			provider.BotTokenEncrypted = &req.BotToken
		}
	}

	if err := s.repo.CreateRepositoryProvider(ctx, provider); err != nil {
		slog.ErrorContext(ctx, "failed to create repository provider",
			"user_id", userID, "provider_type", req.ProviderType, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "repository provider created",
		"user_id", userID, "provider_id", provider.ID, "provider_type", req.ProviderType)
	return provider, nil
}

func (s *Service) GetRepositoryProvider(ctx context.Context, userID, providerID int64) (*user.RepositoryProvider, error) {
	provider, err := s.repo.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return provider, nil
}

func (s *Service) ListRepositoryProviders(ctx context.Context, userID int64) ([]*user.RepositoryProvider, error) {
	return s.repo.ListRepositoryProviders(ctx, userID)
}

func (s *Service) DeleteRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	rowsAffected, err := s.repo.DeleteRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete repository provider",
			"user_id", userID, "provider_id", providerID, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrProviderNotFound
	}
	slog.InfoContext(ctx, "repository provider deleted", "user_id", userID, "provider_id", providerID)
	return nil
}
