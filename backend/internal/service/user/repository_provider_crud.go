package user

import (
	"context"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/crypto"
	"gorm.io/gorm"
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
	var existing user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND name = ?", userID, req.Name).
		First(&existing).Error
	if err == nil {
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
				return nil, err
			}
			provider.ClientSecretEncrypted = &encrypted
		}
		if req.BotToken != "" {
			encrypted, err := crypto.EncryptWithKey(req.BotToken, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			provider.BotTokenEncrypted = &encrypted
		}
	} else {
		// No encryption key - store as-is (not recommended)
		if req.ClientSecret != "" {
			provider.ClientSecretEncrypted = &req.ClientSecret
		}
		if req.BotToken != "" {
			provider.BotTokenEncrypted = &req.BotToken
		}
	}

	if err := s.db.WithContext(ctx).Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// GetRepositoryProvider returns a repository provider by ID
func (s *Service) GetRepositoryProvider(ctx context.Context, userID, providerID int64) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", providerID, userID).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// ListRepositoryProviders returns all repository providers for a user
func (s *Service) ListRepositoryProviders(ctx context.Context, userID int64) ([]*user.RepositoryProvider, error) {
	var providers []*user.RepositoryProvider
	err := s.db.WithContext(ctx).
		Preload("Identity").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&providers).Error
	return providers, err
}

// DeleteRepositoryProvider deletes a repository provider
func (s *Service) DeleteRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", providerID, userID).
		Delete(&user.RepositoryProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}
