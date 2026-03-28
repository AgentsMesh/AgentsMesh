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

// CreateRepositoryProviderRequest represents a request to create a repository provider
type CreateRepositoryProviderRequest struct {
	ProviderType string
	Name         string
	BaseURL      string
	ClientID     string
	ClientSecret string // Plain text, will be encrypted
	BotToken     string // Plain text, will be encrypted
}

// CreateRepositoryProvider creates a new repository provider for a user
func (s *Service) CreateRepositoryProvider(ctx context.Context, userID int64, req *CreateRepositoryProviderRequest) (*user.RepositoryProvider, error) {
	// Validate provider type
	if !user.IsValidProviderType(req.ProviderType) {
		return nil, ErrInvalidProviderType
	}

	// Check if provider with same name already exists
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

	// Set optional fields
	if req.ClientID != "" {
		provider.ClientID = &req.ClientID
	}

	// Encrypt secrets
	if s.encryptionKey != "" {
		if req.ClientSecret != "" {
			encrypted, err := crypto.EncryptWithKey(req.ClientSecret, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt client secret",
					"user_id", userID, "provider_type", req.ProviderType, "error", err)
				return nil, err
			}
			provider.ClientSecretEncrypted = &encrypted
		}
		if req.BotToken != "" {
			encrypted, err := crypto.EncryptWithKey(req.BotToken, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt bot token",
					"user_id", userID, "provider_type", req.ProviderType, "error", err)
				return nil, err
			}
			provider.BotTokenEncrypted = &encrypted
		}
	} else {
		// No encryption key - store as-is (not recommended)
		slog.Warn("storing provider secrets without encryption",
			"user_id", userID, "provider_type", req.ProviderType)
		if req.ClientSecret != "" {
			provider.ClientSecretEncrypted = &req.ClientSecret
		}
		if req.BotToken != "" {
			provider.BotTokenEncrypted = &req.BotToken
		}
	}

	if err := s.repo.CreateRepositoryProvider(ctx, provider); err != nil {
		slog.Error("failed to create repository provider",
			"user_id", userID, "provider_type", req.ProviderType, "error", err)
		return nil, err
	}

	slog.Info("repository provider created",
		"user_id", userID, "provider_id", provider.ID, "provider_type", req.ProviderType)
	return provider, nil
}

// GetRepositoryProvider returns a repository provider by ID
func (s *Service) GetRepositoryProvider(ctx context.Context, userID, providerID int64) (*user.RepositoryProvider, error) {
	provider, err := s.repo.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		if err == user.ErrNotFound {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return provider, nil
}

// ListRepositoryProviders returns all repository providers for a user
func (s *Service) ListRepositoryProviders(ctx context.Context, userID int64) ([]*user.RepositoryProvider, error) {
	return s.repo.ListRepositoryProviders(ctx, userID)
}

// DeleteRepositoryProvider deletes a repository provider
func (s *Service) DeleteRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	rowsAffected, err := s.repo.DeleteRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		slog.Error("failed to delete repository provider",
			"user_id", userID, "provider_id", providerID, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrProviderNotFound
	}
	slog.Info("repository provider deleted", "user_id", userID, "provider_id", providerID)
	return nil
}
