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

func (s *Service) GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error) {
	return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, true)
}

func (s *Service) getOrCreateByOAuthOnce(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string, allowRetry bool) (*user.User, bool, error) {
	identity, err := s.repo.GetIdentityByProviderUser(ctx, provider, providerUserID)
	if err == nil {
		u, err := s.GetByID(ctx, identity.UserID)
		return u, false, err
	}

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
				slog.WarnContext(ctx, "oauth email matches unverified account, using placeholder",
					"provider", provider, "email", email)
			}
		}
	}

	if u == nil {
		userEmail := email
		if userEmail == "" || emailTaken {
			userEmail = fmt.Sprintf("%s_%s@noemail.agentsmesh.placeholder", provider, providerUserID)
		}

		username, err := s.EnsureUniqueUsername(ctx, usernameSeeds(providerUsername, email, name))
		if err != nil {
			slog.ErrorContext(ctx, "failed to derive unique username",
				"provider", provider, "provider_user_id", providerUserID, "error", err)
			return nil, false, err
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
			if allowRetry && isConflictError(err) {
				slog.WarnContext(ctx, "oauth user creation conflict, retrying",
					"provider", provider, "provider_user_id", providerUserID)
				return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, false)
			}
			slog.ErrorContext(ctx, "failed to create oauth user",
				"provider", provider, "provider_user_id", providerUserID, "error", err)
			return nil, false, err
		}
		slog.InfoContext(ctx, "oauth user created",
			"user_id", u.ID, "provider", provider, "provider_user_id", providerUserID)
		isNew = true
	}

	newIdentity := &user.Identity{
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if providerUsername != "" {
		newIdentity.ProviderUsername = &providerUsername
	}

	if err := s.repo.CreateIdentity(ctx, newIdentity); err != nil {
		if allowRetry && isConflictError(err) {
			slog.WarnContext(ctx, "oauth identity creation conflict, retrying",
				"provider", provider, "provider_user_id", providerUserID)
			return s.getOrCreateByOAuthOnce(ctx, provider, providerUserID, providerUsername, email, name, avatarURL, false)
		}
		slog.ErrorContext(ctx, "failed to create oauth identity",
			"user_id", u.ID, "provider", provider, "error", err)
		return nil, false, err
	}

	return u, isNew, nil
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Duplicate entry")
}

// usernameSeeds builds the priority-ordered seed list for EnsureUniqueUsername:
// provider's own username first (most identity-preserving), then email
// local-part, then human name. Empty/garbage seeds are dropped silently and
// EnsureUniqueUsername falls back to a random user-{hex} handle.
func usernameSeeds(providerUsername, email, name string) []string {
	seeds := make([]string, 0, 3)
	if providerUsername != "" {
		seeds = append(seeds, providerUsername)
	}
	if email != "" {
		if local := strings.SplitN(email, "@", 2)[0]; local != "" {
			seeds = append(seeds, local)
		}
	}
	if name != "" {
		seeds = append(seeds, name)
	}
	return seeds
}

func (s *Service) UpdateIdentityTokens(ctx context.Context, userID int64, provider, accessToken, refreshToken string, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"token_expires_at": expiresAt,
	}

	if s.encryptionKey != "" {
		if accessToken != "" {
			encrypted, err := crypto.EncryptWithKey(accessToken, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt oauth access token",
					"user_id", userID, "provider", provider, "error", err)
				return err
			}
			updates["access_token_encrypted"] = encrypted
		}
		if refreshToken != "" {
			encrypted, err := crypto.EncryptWithKey(refreshToken, s.encryptionKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to encrypt oauth refresh token",
					"user_id", userID, "provider", provider, "error", err)
				return err
			}
			updates["refresh_token_encrypted"] = encrypted
		}
	} else {
		slog.WarnContext(ctx, "storing oauth tokens without encryption",
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

func (s *Service) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	return s.repo.GetIdentity(ctx, userID, provider)
}

func (s *Service) GetIdentityByProvider(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	return s.GetIdentity(ctx, userID, provider)
}

type DecryptedTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}

func (s *Service) GetDecryptedTokens(ctx context.Context, userID int64, provider string) (*DecryptedTokens, error) {
	identity, err := s.GetIdentity(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	tokens := &DecryptedTokens{
		ExpiresAt: identity.TokenExpiresAt,
	}

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
		if identity.AccessTokenEncrypted != nil {
			tokens.AccessToken = *identity.AccessTokenEncrypted
		}
		if identity.RefreshTokenEncrypted != nil {
			tokens.RefreshToken = *identity.RefreshTokenEncrypted
		}
	}

	return tokens, nil
}

func (s *Service) ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error) {
	return s.repo.ListIdentities(ctx, userID)
}

func (s *Service) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	slog.InfoContext(ctx, "deleting oauth identity", "user_id", userID, "provider", provider)
	return s.repo.DeleteIdentity(ctx, userID, provider)
}
