package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

// oauthStateKeyPrefix is the Redis key prefix for OAuth states
const oauthStateKeyPrefix = "oauth:state:"

// oauthStateTTL is the expiration time for OAuth states (10 minutes)
const oauthStateTTL = 10 * time.Minute

// GetOAuthURL returns the OAuth authorization URL
func (s *Service) GetOAuthURL(provider, state string) (string, error) {
	cfg, ok := s.config.OAuthProviders[provider]
	if !ok {
		return "", errors.New("unsupported OAuth provider")
	}

	switch provider {
	case "github":
		return getGitHubAuthURL(cfg, state), nil
	case "google":
		return getGoogleAuthURL(cfg, state), nil
	case "gitlab":
		return getGitLabAuthURL(cfg, state), nil
	case "gitee":
		return getGiteeAuthURL(cfg, state), nil
	default:
		return "", errors.New("unsupported OAuth provider")
	}
}

// HandleOAuthCallback handles OAuth callback
func (s *Service) HandleOAuthCallback(ctx context.Context, provider, code, state string) (*user.User, *TokenPair, bool, error) {
	cfg, ok := s.config.OAuthProviders[provider]
	if !ok {
		return nil, nil, false, errors.New("unsupported OAuth provider")
	}

	var userInfo *OAuthUserInfo
	var err error

	switch provider {
	case "github":
		userInfo, err = handleGitHubCallback(ctx, cfg, code)
	case "google":
		userInfo, err = handleGoogleCallback(ctx, cfg, code)
	case "gitlab":
		userInfo, err = handleGitLabCallback(ctx, cfg, code)
	case "gitee":
		userInfo, err = handleGiteeCallback(ctx, cfg, code)
	default:
		return nil, nil, false, errors.New("unsupported OAuth provider")
	}

	if err != nil {
		return nil, nil, false, err
	}

	// Get or create user
	u, isNew, err := s.userService.GetOrCreateByOAuth(ctx, provider, userInfo.ID, userInfo.Username, userInfo.Email, userInfo.Name, userInfo.AvatarURL)
	if err != nil {
		return nil, nil, false, err
	}

	// Save OAuth access token to identity for later API calls
	if userInfo.AccessToken != "" {
		if err := s.userService.UpdateIdentityTokens(ctx, u.ID, provider, userInfo.AccessToken, "", nil); err != nil {
			slog.Warn("failed to save OAuth token",
				"user_id", u.ID,
				"provider", provider,
				"error", err,
			)
		}
	}

	// For Git providers, ensure a RepositoryProvider exists
	if provider == "github" || provider == "gitlab" || provider == "gitee" {
		if err := s.userService.EnsureRepositoryProviderForIdentity(ctx, u.ID, provider); err != nil {
			slog.Warn("failed to create repository provider",
				"user_id", u.ID,
				"provider", provider,
				"error", err,
			)
		}
	}

	// Generate tokens
	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, nil, false, err
	}

	return u, tokens, isNew, nil
}

// GenerateOAuthState generates and stores OAuth state in Redis
func (s *Service) GenerateOAuthState(ctx context.Context, provider, redirectURL string) (string, error) {
	state, err := GenerateState()
	if err != nil {
		return "", err
	}

	key := oauthStateKeyPrefix + state
	if err := s.redis.Set(ctx, key, redirectURL, oauthStateTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	return state, nil
}

// ValidateOAuthState validates OAuth state and returns redirect URL
func (s *Service) ValidateOAuthState(ctx context.Context, state string) (string, error) {
	key := oauthStateKeyPrefix + state

	redirectURL, err := s.redis.GetDel(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return "", ErrInvalidState
		}
		return "", fmt.Errorf("failed to validate OAuth state: %w", err)
	}

	return redirectURL, nil
}

// OAuthLogin handles OAuth login
func (s *Service) OAuthLogin(ctx context.Context, req *OAuthLoginRequest) (*LoginResult, error) {
	u, _, err := s.userService.GetOrCreateByOAuth(ctx, req.Provider, req.ProviderUserID, req.Username, req.Email, req.Name, req.AvatarURL)
	if err != nil {
		return nil, err
	}

	if req.AccessToken != "" {
		s.userService.UpdateIdentityTokens(ctx, u.ID, req.Provider, req.AccessToken, req.RefreshToken, req.ExpiresAt)
	}

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}
