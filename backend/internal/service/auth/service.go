package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/user"
	userService "github.com/anthropics/agentmesh/backend/internal/service/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrRefreshExpired      = errors.New("refresh token expired")
	ErrInvalidOAuthCode    = errors.New("invalid OAuth code")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserDisabled        = errors.New("user is disabled")
	ErrEmailExists         = errors.New("email already exists")
	ErrUsernameExists      = errors.New("username already exists")
	ErrInvalidState        = errors.New("invalid OAuth state")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// Redis key prefixes
const (
	refreshTokenPrefix = "auth:refresh:"   // Stores refresh token data
	tokenBlacklistKey  = "auth:blacklist:" // Stores revoked access tokens
)

// Config holds auth configuration
type Config struct {
	JWTSecret         string
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration
	Issuer            string
	OAuthProviders    map[string]OAuthConfig
}

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Claims represents JWT claims
type Claims struct {
	UserID         int64  `json:"user_id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	OrganizationID int64  `json:"organization_id,omitempty"`
	Role           string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// RefreshTokenData stores refresh token metadata
type RefreshTokenData struct {
	UserID         int64     `json:"user_id"`
	OrganizationID int64     `json:"organization_id,omitempty"`
	Role           string    `json:"role,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ID          string
	Username    string
	Email       string
	Name        string
	AvatarURL   string
	AccessToken string // OAuth access token for API calls
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string
	Username string
	Password string
	Name     string
}

// LoginResult represents the result of a login operation
type LoginResult struct {
	User         *user.User
	Token        string
	RefreshToken string
	ExpiresIn    int64
}

// OAuthLoginRequest represents OAuth login request
type OAuthLoginRequest struct {
	Provider       string
	ProviderUserID string
	Email          string
	Username       string
	Name           string
	AvatarURL      string
	AccessToken    string
	RefreshToken   string
	ExpiresAt      *time.Time
}

// oauthStateKeyPrefix is the Redis key prefix for OAuth states
const oauthStateKeyPrefix = "oauth:state:"

// oauthStateTTL is the expiration time for OAuth states (10 minutes)
const oauthStateTTL = 10 * time.Minute

// Service handles authentication
type Service struct {
	config      *Config
	userService *userService.Service
	redis       *redis.Client
}

// NewService creates a new auth service
func NewService(cfg *Config, userSvc *userService.Service) *Service {
	return &Service{
		config:      cfg,
		userService: userSvc,
	}
}

// NewServiceWithRedis creates a new auth service with Redis support
func NewServiceWithRedis(cfg *Config, userSvc *userService.Service, redisClient *redis.Client) *Service {
	return &Service{
		config:      cfg,
		userService: userSvc,
		redis:       redisClient,
	}
}

// Login authenticates user and returns tokens
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := s.userService.Authenticate(ctx, email, password)
	if err != nil {
		if err == userService.ErrInvalidCredentials {
			return nil, ErrInvalidCredentials
		}
		if err == userService.ErrUserInactive {
			return nil, ErrUserDisabled
		}
		return nil, err
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

// Register creates a new user and returns tokens
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*LoginResult, error) {
	u, err := s.userService.Create(ctx, &userService.CreateRequest{
		Email:    req.Email,
		Username: req.Username,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		if err == userService.ErrEmailAlreadyExists {
			return nil, ErrEmailExists
		}
		if err == userService.ErrUsernameExists {
			return nil, ErrUsernameExists
		}
		return nil, err
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
