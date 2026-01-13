package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	JWTSecret          string
	JWTExpiration      time.Duration
	RefreshExpiration  time.Duration
	Issuer             string
	OAuthProviders     map[string]OAuthConfig
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

// GenerateTokenPair generates access and refresh tokens
func (s *Service) GenerateTokenPair(u *user.User, orgID int64, role string) (*TokenPair, error) {
	return s.GenerateTokenPairWithContext(context.Background(), u, orgID, role)
}

// GenerateTokenPairWithContext generates access and refresh tokens with context
func (s *Service) GenerateTokenPairWithContext(ctx context.Context, u *user.User, orgID int64, role string) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.JWTExpiration)
	refreshExpiresAt := now.Add(s.config.RefreshExpiration)

	claims := &Claims{
		UserID:         u.ID,
		Email:          u.Email,
		Username:       u.Username,
		OrganizationID: orgID,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			Subject:   u.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, err
	}
	refreshToken := base64.URLEncoding.EncodeToString(refreshBytes)

	// Store refresh token in Redis if available
	if s.redis != nil {
		tokenData := &RefreshTokenData{
			UserID:         u.ID,
			OrganizationID: orgID,
			Role:           role,
			CreatedAt:      now,
			ExpiresAt:      refreshExpiresAt,
		}
		if err := s.storeRefreshToken(ctx, refreshToken, tokenData); err != nil {
			return nil, fmt.Errorf("failed to store refresh token: %w", err)
		}
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// storeRefreshToken stores refresh token data in Redis
func (s *Service) storeRefreshToken(ctx context.Context, refreshToken string, data *RefreshTokenData) error {
	// Hash the token for storage (security best practice)
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ttl := time.Until(data.ExpiresAt)
	return s.redis.Set(ctx, key, jsonData, ttl).Err()
}

// hashToken creates a SHA-256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateToken validates a JWT token
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.ValidateTokenWithContext(context.Background(), tokenString)
}

// ValidateTokenWithContext validates a JWT token with context and blacklist check
func (s *Service) ValidateTokenWithContext(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is blacklisted (revoked)
	if s.redis != nil {
		if revoked, _ := s.isTokenBlacklisted(ctx, tokenString); revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

// isTokenBlacklisted checks if a token has been revoked
func (s *Service) isTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	tokenHash := hashToken(token)
	key := tokenBlacklistKey + tokenHash
	exists, err := s.redis.Exists(ctx, key).Result()
	return exists > 0, err
}

// RefreshTokens generates new tokens using refresh token
func (s *Service) RefreshTokens(ctx context.Context, accessToken, refreshToken string) (*TokenPair, error) {
	// Parse expired access token to get user info
	token, _ := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Get user
	u, err := s.userService.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new token pair
	return s.GenerateTokenPair(u, claims.OrganizationID, claims.Role)
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

	// Generate tokens
	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, nil, false, err
	}

	return u, tokens, isNew, nil
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ID        string
	Username  string
	Email     string
	Name      string
	AvatarURL string
}

// GitHub OAuth helpers
func getGitHubAuthURL(cfg OAuthConfig, state string) string {
	return "https://github.com/login/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&scope=user:email" +
		"&state=" + state
}

func handleGitHubCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	// Exchange code for access token (use client with timeout)
	client := &http.Client{Timeout: 30 * time.Second}
	tokenResp, err := client.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	body, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// Parse access_token from response (format: access_token=xxx&token_type=bearer&scope=...)
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken := values.Get("access_token")
	if accessToken == "" {
		return nil, ErrInvalidOAuthCode
	}

	// Get user info from GitHub API
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", userResp.StatusCode)
	}

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is empty, fetch from emails endpoint (private emails)
	email := ghUser.Email
	if email == "" {
		email, _ = getGitHubPrimaryEmail(ctx, accessToken)
	}

	return &OAuthUserInfo{
		ID:        fmt.Sprintf("%d", ghUser.ID),
		Username:  ghUser.Login,
		Email:     email,
		Name:      ghUser.Name,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}

// getGitHubPrimaryEmail fetches primary email from GitHub emails endpoint
func getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}

// Google OAuth helpers
func getGoogleAuthURL(cfg OAuthConfig, state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=email profile" +
		"&state=" + state
}

func handleGoogleCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	// Exchange code for access token
	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		return nil, ErrInvalidOAuthCode
	}

	// Get user info from Google API
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", userResp.StatusCode)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&googleUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Generate username from email if not provided
	username := strings.Split(googleUser.Email, "@")[0]

	return &OAuthUserInfo{
		ID:        googleUser.ID,
		Username:  username,
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		AvatarURL: googleUser.Picture,
	}, nil
}

// GitLab OAuth helpers
func getGitLabAuthURL(cfg OAuthConfig, state string) string {
	return "https://gitlab.com/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=read_user" +
		"&state=" + state
}

func handleGitLabCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	// Exchange code for access token
	tokenResp, err := http.PostForm("https://gitlab.com/oauth/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		return nil, ErrInvalidOAuthCode
	}

	// Get user info from GitLab API
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitlab.com/api/v4/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", userResp.StatusCode)
	}

	var glUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&glUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:        fmt.Sprintf("%d", glUser.ID),
		Username:  glUser.Username,
		Email:     glUser.Email,
		Name:      glUser.Name,
		AvatarURL: glUser.AvatarURL,
	}, nil
}

// Gitee OAuth helpers
func getGiteeAuthURL(cfg OAuthConfig, state string) string {
	return "https://gitee.com/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=user_info" +
		"&state=" + state
}

func handleGiteeCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	// Exchange code for access token
	tokenResp, err := http.PostForm("https://gitee.com/oauth/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		return nil, ErrInvalidOAuthCode
	}

	// Get user info from Gitee API
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitee.com/api/v5/user?access_token="+tokenData.AccessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gitee API returned status %d", userResp.StatusCode)
	}

	var giteeUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&giteeUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:        fmt.Sprintf("%d", giteeUser.ID),
		Username:  giteeUser.Login,
		Email:     giteeUser.Email,
		Name:      giteeUser.Name,
		AvatarURL: giteeUser.AvatarURL,
	}, nil
}

// GenerateState generates a random state for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
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

// GenerateTokens generates tokens for a user (used after email verification)
func (s *Service) GenerateTokens(ctx context.Context, u *user.User) (*LoginResult, error) {
	tokens, err := s.GenerateTokenPairWithContext(ctx, u, 0, "")
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

// GenerateOAuthState generates and stores OAuth state in Redis
func (s *Service) GenerateOAuthState(ctx context.Context, provider, redirectURL string) (string, error) {
	state, err := GenerateState()
	if err != nil {
		return "", err
	}

	// Store state in Redis with TTL
	key := oauthStateKeyPrefix + state
	if err := s.redis.Set(ctx, key, redirectURL, oauthStateTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	return state, nil
}

// ValidateOAuthState validates OAuth state and returns redirect URL
func (s *Service) ValidateOAuthState(ctx context.Context, state string) (string, error) {
	key := oauthStateKeyPrefix + state

	// Get and delete state atomically
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
	// Get or create user
	u, _, err := s.userService.GetOrCreateByOAuth(ctx, req.Provider, req.ProviderUserID, req.Username, req.Email, req.Name, req.AvatarURL)
	if err != nil {
		return nil, err
	}

	// Update identity tokens
	if req.AccessToken != "" {
		s.userService.UpdateIdentityTokens(ctx, u.ID, req.Provider, req.AccessToken, req.RefreshToken, req.ExpiresAt)
	}

	// Generate tokens
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

// RefreshToken refreshes access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	// If Redis is not available, fall back to stateless validation
	if s.redis == nil {
		return nil, ErrInvalidRefreshToken
	}

	// Validate and get refresh token data from Redis
	tokenData, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	u, err := s.userService.GetByID(ctx, tokenData.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user is active
	if !u.IsActive {
		return nil, ErrUserDisabled
	}

	// Invalidate old refresh token (token rotation for security)
	if err := s.revokeRefreshToken(ctx, refreshToken); err != nil {
		// Log but don't fail - new token will still be issued
		// In production, you might want to log this error
	}

	// Generate new tokens
	tokens, err := s.GenerateTokenPairWithContext(ctx, u, tokenData.OrganizationID, tokenData.Role)
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

// validateRefreshToken validates a refresh token against Redis storage
func (s *Service) validateRefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenData, error) {
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash

	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	var tokenData RefreshTokenData
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token data: %w", err)
	}

	// Check if token has expired
	if time.Now().After(tokenData.ExpiresAt) {
		// Clean up expired token
		s.redis.Del(ctx, key)
		return nil, ErrRefreshExpired
	}

	return &tokenData, nil
}

// revokeRefreshToken removes a refresh token from Redis
func (s *Service) revokeRefreshToken(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash
	return s.redis.Del(ctx, key).Err()
}

// RevokeToken revokes an access token by adding it to the blacklist
func (s *Service) RevokeToken(ctx context.Context, token string) error {
	// If Redis is not available, we can't maintain a blacklist
	if s.redis == nil {
		return nil
	}

	// Parse the token to get expiration time
	claims, err := s.ValidateToken(token)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		// If token is invalid (not just expired), nothing to revoke
		return nil
	}

	// Calculate TTL - token should be in blacklist until it would have expired naturally
	var ttl time.Duration
	if claims != nil && claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			// Token already expired, no need to blacklist
			return nil
		}
	} else {
		// Fallback to JWT expiration duration if we can't parse claims
		ttl = s.config.JWTExpiration
	}

	// Add token to blacklist
	tokenHash := hashToken(token)
	key := tokenBlacklistKey + tokenHash
	return s.redis.Set(ctx, key, "1", ttl).Err()
}

// RevokeAllUserTokens revokes all tokens for a user (useful for password change, account compromise, etc.)
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	if s.redis == nil {
		return nil
	}

	// Delete all refresh tokens for this user
	// Note: This is a simplified implementation. In production, you might want to
	// store a user->tokens mapping for efficient revocation
	pattern := refreshTokenPrefix + "*"
	iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var tokenData RefreshTokenData
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			continue
		}
		if tokenData.UserID == userID {
			s.redis.Del(ctx, key)
		}
	}
	return iter.Err()
}
