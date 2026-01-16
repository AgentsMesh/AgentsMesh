package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
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

// UpdateRepositoryProviderRequest represents a request to update a repository provider
type UpdateRepositoryProviderRequest struct {
	Name         *string
	BaseURL      *string
	ClientID     *string
	ClientSecret *string // Plain text, will be encrypted
	BotToken     *string // Plain text, will be encrypted
	IsActive     *bool
}

// UpdateRepositoryProvider updates a repository provider
func (s *Service) UpdateRepositoryProvider(ctx context.Context, userID, providerID int64, req *UpdateRepositoryProviderRequest) (*user.RepositoryProvider, error) {
	// Verify ownership
	provider, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != "" {
		// Check if new name conflicts with existing provider
		var existing user.RepositoryProvider
		err := s.db.WithContext(ctx).
			Where("user_id = ? AND name = ? AND id != ?", userID, *req.Name, providerID).
			First(&existing).Error
		if err == nil {
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

	// Handle secret encryption
	if req.ClientSecret != nil {
		if *req.ClientSecret == "" {
			updates["client_secret_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.ClientSecret, s.encryptionKey)
			if err != nil {
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

	if err := s.db.WithContext(ctx).Model(provider).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetRepositoryProvider(ctx, userID, providerID)
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

// SetDefaultRepositoryProvider sets a repository provider as default
func (s *Service) SetDefaultRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	// Verify ownership
	_, err := s.GetRepositoryProvider(ctx, userID, providerID)
	if err != nil {
		return err
	}

	// Start transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all defaults for this user
		if err := tx.Model(&user.RepositoryProvider{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		return tx.Model(&user.RepositoryProvider{}).
			Where("id = ? AND user_id = ?", providerID, userID).
			Update("is_default", true).Error
	})
}

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
