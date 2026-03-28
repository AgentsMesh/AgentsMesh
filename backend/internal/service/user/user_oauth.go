package user

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// GetOrCreateByOAuth gets or creates a user from OAuth identity.
// Uses retry-on-conflict to handle concurrent SSO callbacks for the same user.
func (s *Service) GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error) {
	return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, true)
}

func (s *Service) getOrCreateByOAuthOnce(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string, allowRetry bool) (*user.User, bool, error) {
	// Check if identity already exists
	identity, err := s.repo.GetIdentityByProviderUser(ctx, provider, providerUserID)
	if err == nil {
		// Identity exists, get user
		u, err := s.GetByID(ctx, identity.UserID)
		return u, false, err
	}

	// Check if user with email exists (only when email is non-empty to avoid
	// matching unrelated users who also have empty emails).
	// Only link to existing users who have verified their email to prevent
	// OAuth email takeover attacks (e.g., attacker registers on OAuth provider
	// with victim's email, then logs in to take over their account).
	var u *user.User
	var isNew bool
	var emailTaken bool // whether email already exists in DB (verified or not)
	if email != "" {
		existing, err := s.GetByEmail(ctx, email)
		if err == nil {
			if existing.IsEmailVerified {
				u = existing
			} else {
				emailTaken = true
				slog.Warn("oauth email matches unverified account, using placeholder",
					"provider", provider, "email", email)
			}
		}
	}

	if u == nil {
		// Create new user.
		// Use a placeholder email when:
		// - OAuth provider returned no email, OR
		// - the email is already taken by an unverified account
		// The email column has a unique constraint.
		userEmail := email
		if userEmail == "" || emailTaken {
			userEmail = fmt.Sprintf("%s_%s@noemail.agentsmesh.placeholder", provider, providerUserID)
		}

		username := providerUsername
		if username == "" {
			username = email
		}

		// Ensure username is unique
		for i := 0; i < 100; i++ {
			if _, err := s.GetByUsername(ctx, username); err != nil {
				break
			}
			username = fmt.Sprintf("%s_%d", providerUsername, i)
		}

		u = &user.User{
			Email:    userEmail,
			Username: username,
			IsActive: true,
		}
		if name != "" {
			u.Name = &name
		}
		if avatarURL != "" {
			u.AvatarURL = &avatarURL
		}

		if err := s.repo.CreateUser(ctx, u); err != nil {
			// Concurrent SSO callback may have created the same user — retry once
			if allowRetry && isConflictError(err) {
				slog.Warn("oauth user creation conflict, retrying",
					"provider", provider, "provider_user_id", providerUserID)
				return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, false)
			}
			slog.Error("failed to create oauth user",
				"provider", provider, "provider_user_id", providerUserID, "error", err)
			return nil, false, err
		}
		slog.Info("oauth user created",
			"user_id", u.ID, "provider", provider, "provider_user_id", providerUserID)
		isNew = true
	}

	// Create identity
	newIdentity := &user.Identity{
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if providerUsername != "" {
		newIdentity.ProviderUsername = &providerUsername
	}

	if err := s.repo.CreateIdentity(ctx, newIdentity); err != nil {
		// Concurrent SSO callback may have created the same identity — retry once
		if allowRetry && isConflictError(err) {
			slog.Warn("oauth identity creation conflict, retrying",
				"provider", provider, "provider_user_id", providerUserID)
			return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, false)
		}
		slog.Error("failed to create oauth identity",
			"user_id", u.ID, "provider", provider, "error", err)
		return nil, false, err
	}

	return u, isNew, nil
}

// isConflictError checks if err is a unique constraint violation (PostgreSQL/SQLite/MySQL).
func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Duplicate entry")
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
				slog.Error("failed to encrypt oauth access token",
					"user_id", userID, "provider", provider, "error", err)
				return err
			}
			updates["access_token_encrypted"] = encrypted
		}
		if refreshToken != "" {
			encrypted, err := crypto.EncryptWithKey(refreshToken, s.encryptionKey)
			if err != nil {
				slog.Error("failed to encrypt oauth refresh token",
					"user_id", userID, "provider", provider, "error", err)
				return err
			}
			updates["refresh_token_encrypted"] = encrypted
		}
	} else {
		// Fallback: store as-is (not recommended for production)
		slog.Warn("storing oauth tokens without encryption",
			"user_id", userID, "provider", provider)
		if accessToken != "" {
			updates["access_token_encrypted"] = accessToken
		}
		if refreshToken != "" {
			updates["refresh_token_encrypted"] = refreshToken
		}
	}

	return s.repo.UpdateIdentityFields(ctx, userID, provider, updates)
}

// GetIdentity returns an OAuth identity
func (s *Service) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	return s.repo.GetIdentity(ctx, userID, provider)
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
	return s.repo.ListIdentities(ctx, userID)
}

// DeleteIdentity deletes an OAuth identity
func (s *Service) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	slog.Info("deleting oauth identity", "user_id", userID, "provider", provider)
	return s.repo.DeleteIdentity(ctx, userID, provider)
}
