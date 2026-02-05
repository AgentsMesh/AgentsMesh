package auth

import (
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
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
