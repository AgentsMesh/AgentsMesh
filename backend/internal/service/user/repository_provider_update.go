package user

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/crypto"
	"gorm.io/gorm"
)

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
