package user

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/crypto"
)

// GetOrCreateByOAuth gets or creates a user from OAuth identity
func (s *Service) GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error) {
	// Check if identity already exists
	var identity user.Identity
	if err := s.db.WithContext(ctx).Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&identity).Error; err == nil {
		// Identity exists, get user
		u, err := s.GetByID(ctx, identity.UserID)
		return u, false, err
	}

	// Check if user with email exists
	var u *user.User
	var isNew bool
	existing, err := s.GetByEmail(ctx, email)
	if err == nil {
		u = existing
	} else {
		// Create new user
		username := providerUsername
		if username == "" {
			username = email
		}

		// Ensure username is unique
		for i := 0; i < 100; i++ {
			if _, err := s.GetByUsername(ctx, username); err != nil {
				break
			}
			username = providerUsername + "_" + string(rune('0'+i))
		}

		u = &user.User{
			Email:    email,
			Username: username,
			IsActive: true,
		}
		if name != "" {
			u.Name = &name
		}
		if avatarURL != "" {
			u.AvatarURL = &avatarURL
		}

		if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
			return nil, false, err
		}
		isNew = true
	}

	// Create identity
	identity = user.Identity{
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if providerUsername != "" {
		identity.ProviderUsername = &providerUsername
	}

	if err := s.db.WithContext(ctx).Create(&identity).Error; err != nil {
		return nil, false, err
	}

	return u, isNew, nil
}

// UpdateIdentityTokens updates OAuth tokens for an identity
// Tokens are encrypted using AES-GCM before storage
func (s *Service) UpdateIdentityTokens(ctx context.Context, userID int64, provider, accessToken, refreshToken string, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"token_expires_at": expiresAt,
	}

	// Encrypt tokens if encryption key is configured
	if s.encryptionKey != "" {
		if accessToken != "" {
			encrypted, err := crypto.EncryptWithKey(accessToken, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["access_token_encrypted"] = encrypted
		}
		if refreshToken != "" {
			encrypted, err := crypto.EncryptWithKey(refreshToken, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["refresh_token_encrypted"] = encrypted
		}
	} else {
		// Fallback: store as-is (not recommended for production)
		if accessToken != "" {
			updates["access_token_encrypted"] = accessToken
		}
		if refreshToken != "" {
			updates["refresh_token_encrypted"] = refreshToken
		}
	}

	return s.db.WithContext(ctx).Model(&user.Identity{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Updates(updates).Error
}

// GetIdentity returns an OAuth identity
func (s *Service) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	var identity user.Identity
	if err := s.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error; err != nil {
		return nil, err
	}
	return &identity, nil
}

// GetIdentityByProvider returns an OAuth identity by provider (alias for GetIdentity)
func (s *Service) GetIdentityByProvider(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	return s.GetIdentity(ctx, userID, provider)
}

// DecryptedTokens holds decrypted OAuth tokens
type DecryptedTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}

// GetDecryptedTokens retrieves and decrypts OAuth tokens for an identity
func (s *Service) GetDecryptedTokens(ctx context.Context, userID int64, provider string) (*DecryptedTokens, error) {
	identity, err := s.GetIdentity(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	tokens := &DecryptedTokens{
		ExpiresAt: identity.TokenExpiresAt,
	}

	// Decrypt tokens if encryption key is configured
	if s.encryptionKey != "" {
		if identity.AccessTokenEncrypted != nil && *identity.AccessTokenEncrypted != "" {
			decrypted, err := crypto.DecryptWithKey(*identity.AccessTokenEncrypted, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			tokens.AccessToken = decrypted
		}
		if identity.RefreshTokenEncrypted != nil && *identity.RefreshTokenEncrypted != "" {
			decrypted, err := crypto.DecryptWithKey(*identity.RefreshTokenEncrypted, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			tokens.RefreshToken = decrypted
		}
	} else {
		// No encryption key - return as-is
		if identity.AccessTokenEncrypted != nil {
			tokens.AccessToken = *identity.AccessTokenEncrypted
		}
		if identity.RefreshTokenEncrypted != nil {
			tokens.RefreshToken = *identity.RefreshTokenEncrypted
		}
	}

	return tokens, nil
}

// ListIdentities returns all identities for a user
func (s *Service) ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error) {
	var identities []*user.Identity
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

// DeleteIdentity deletes an OAuth identity
func (s *Service) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).Delete(&user.Identity{}).Error
}
