package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Mock user for testing
func createMockUser() *user.User {
	name := "Test User"
	return &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Name:     &name,
		IsActive: true,
	}
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			name TEXT,
			avatar_url TEXT,
			password_hash TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			last_login_at DATETIME,
			is_email_verified INTEGER NOT NULL DEFAULT 0,
			email_verification_token TEXT,
			email_verification_expires_at DATETIME,
			password_reset_token TEXT,
			password_reset_expires_at DATETIME,
			default_git_credential_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_identities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider TEXT NOT NULL,
			provider_user_id TEXT NOT NULL,
			provider_username TEXT,
			access_token_encrypted TEXT,
			refresh_token_encrypted TEXT,
			token_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create user_identities table: %v", err)
	}

	return db
}

func TestNewService(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "test-secret",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, nil)
	if svc == nil {
		t.Error("NewService returned nil")
	}
	if svc.config != cfg {
		t.Error("Service config not set correctly")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, nil)
	mockUser := createMockUser()

	tests := []struct {
		name  string
		user  *user.User
		orgID int64
		role  string
	}{
		{
			name:  "basic token generation",
			user:  mockUser,
			orgID: 0,
			role:  "",
		},
		{
			name:  "token with org and role",
			user:  mockUser,
			orgID: 123,
			role:  "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := svc.GenerateTokenPair(tt.user, tt.orgID, tt.role)
			if err != nil {
				t.Fatalf("GenerateTokenPair failed: %v", err)
			}

			if tokens.AccessToken == "" {
				t.Error("AccessToken is empty")
			}
			if tokens.RefreshToken == "" {
				t.Error("RefreshToken is empty")
			}
			if tokens.TokenType != "Bearer" {
				t.Errorf("TokenType = %s, want Bearer", tokens.TokenType)
			}
			if tokens.ExpiresAt.Before(time.Now()) {
				t.Error("ExpiresAt should be in the future")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, nil)
	mockUser := createMockUser()

	t.Run("valid token", func(t *testing.T) {
		tokens, err := svc.GenerateTokenPair(mockUser, 123, "admin")
		if err != nil {
			t.Fatalf("Failed to generate tokens: %v", err)
		}

		claims, err := svc.ValidateToken(tokens.AccessToken)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}

		if claims.UserID != mockUser.ID {
			t.Errorf("UserID = %d, want %d", claims.UserID, mockUser.ID)
		}
		if claims.Email != mockUser.Email {
			t.Errorf("Email = %s, want %s", claims.Email, mockUser.Email)
		}
		if claims.Username != mockUser.Username {
			t.Errorf("Username = %s, want %s", claims.Username, mockUser.Username)
		}
		if claims.OrganizationID != 123 {
			t.Errorf("OrganizationID = %d, want 123", claims.OrganizationID)
		}
		if claims.Role != "admin" {
			t.Errorf("Role = %s, want admin", claims.Role)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ValidateToken("invalid-token")
		if err == nil {
			t.Error("Expected error for invalid token")
		}
		if err != ErrInvalidToken {
			t.Errorf("Expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := svc.ValidateToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.invalid")
		if err == nil {
			t.Error("Expected error for malformed token")
		}
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		otherCfg := &Config{
			JWTSecret:     "different-secret-key-at-least-32-bytes",
			JWTExpiration: time.Hour,
			Issuer:        "test-issuer",
		}
		otherSvc := NewService(otherCfg, nil)
		tokens, _ := otherSvc.GenerateTokenPair(mockUser, 0, "")

		_, err := svc.ValidateToken(tokens.AccessToken)
		if err == nil {
			t.Error("Expected error for token with wrong secret")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expiredCfg := &Config{
			JWTSecret:     "test-secret-key-at-least-32-bytes",
			JWTExpiration: -time.Hour, // Negative duration = already expired
			Issuer:        "test-issuer",
		}
		expiredSvc := NewService(expiredCfg, nil)
		tokens, _ := expiredSvc.GenerateTokenPair(mockUser, 0, "")

		_, err := svc.ValidateToken(tokens.AccessToken)
		if err == nil {
			t.Error("Expected error for expired token")
		}
		if err != ErrTokenExpired {
			t.Errorf("Expected ErrTokenExpired, got %v", err)
		}
	})
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}
	if state1 == "" {
		t.Error("GenerateState returned empty string")
	}

	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState failed: %v", err)
	}

	// States should be unique
	if state1 == state2 {
		t.Error("GenerateState returned duplicate states")
	}
}

func TestGenerateOAuthState(t *testing.T) {
	// Start miniredis for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
	}
	svc := NewServiceWithRedis(cfg, nil, redisClient)
	ctx := context.Background()

	state, err := svc.GenerateOAuthState(ctx, "github", "https://example.com/callback")
	if err != nil {
		t.Fatalf("GenerateOAuthState failed: %v", err)
	}
	if state == "" {
		t.Error("State is empty")
	}

	// Validate the state
	redirectURL, err := svc.ValidateOAuthState(ctx, state)
	if err != nil {
		t.Fatalf("ValidateOAuthState failed: %v", err)
	}
	if redirectURL != "https://example.com/callback" {
		t.Errorf("RedirectURL = %s, want https://example.com/callback", redirectURL)
	}
}

func TestValidateOAuthState(t *testing.T) {
	// Start miniredis for testing
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
	}
	svc := NewServiceWithRedis(cfg, nil, redisClient)
	ctx := context.Background()

	t.Run("valid state", func(t *testing.T) {
		state, _ := svc.GenerateOAuthState(ctx, "github", "https://example.com")
		redirectURL, err := svc.ValidateOAuthState(ctx, state)
		if err != nil {
			t.Fatalf("ValidateOAuthState failed: %v", err)
		}
		if redirectURL != "https://example.com" {
			t.Errorf("RedirectURL mismatch")
		}
	})

	t.Run("invalid state", func(t *testing.T) {
		_, err := svc.ValidateOAuthState(ctx, "invalid-state")
		if err == nil {
			t.Error("Expected error for invalid state")
		}
		if err != ErrInvalidState {
			t.Errorf("Expected ErrInvalidState, got %v", err)
		}
	})

	t.Run("state used twice", func(t *testing.T) {
		state, _ := svc.GenerateOAuthState(ctx, "github", "https://example.com")
		// First use should succeed
		_, err := svc.ValidateOAuthState(ctx, state)
		if err != nil {
			t.Fatalf("First validation failed: %v", err)
		}
		// Second use should fail (state is deleted after first use)
		_, err = svc.ValidateOAuthState(ctx, state)
		if err == nil {
			t.Error("Expected error for reused state")
		}
	})
}

func TestGetOAuthURL(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
		OAuthProviders: map[string]OAuthConfig{
			"github": {
				ClientID:     "github-client-id",
				ClientSecret: "github-secret",
				RedirectURL:  "https://example.com/callback/github",
				Scopes:       []string{"user:email"},
			},
			"google": {
				ClientID:     "google-client-id",
				ClientSecret: "google-secret",
				RedirectURL:  "https://example.com/callback/google",
				Scopes:       []string{"email", "profile"},
			},
			"gitlab": {
				ClientID:     "gitlab-client-id",
				ClientSecret: "gitlab-secret",
				RedirectURL:  "https://example.com/callback/gitlab",
				Scopes:       []string{"read_user"},
			},
			"gitee": {
				ClientID:     "gitee-client-id",
				ClientSecret: "gitee-secret",
				RedirectURL:  "https://example.com/callback/gitee",
				Scopes:       []string{"user_info"},
			},
		},
	}
	svc := NewService(cfg, nil)

	tests := []struct {
		provider    string
		expectError bool
		contains    string
	}{
		{"github", false, "github.com/login/oauth/authorize"},
		{"google", false, "accounts.google.com/o/oauth2/v2/auth"},
		{"gitlab", false, "gitlab.com/oauth/authorize"},
		{"gitee", false, "gitee.com/oauth/authorize"},
		{"unsupported", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			url, err := svc.GetOAuthURL(tt.provider, "test-state")
			if tt.expectError {
				if err == nil {
					t.Error("Expected error for unsupported provider")
				}
			} else {
				if err != nil {
					t.Fatalf("GetOAuthURL failed: %v", err)
				}
				if url == "" {
					t.Error("URL is empty")
				}
				if tt.contains != "" && !contains(url, tt.contains) {
					t.Errorf("URL does not contain %s: %s", tt.contains, url)
				}
			}
		})
	}
}

func TestOAuthURLHelpers(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"user:email"},
	}

	tests := []struct {
		name     string
		fn       func(OAuthConfig, string) string
		contains string
	}{
		{"GitHub", getGitHubAuthURL, "github.com"},
		{"Google", getGoogleAuthURL, "accounts.google.com"},
		{"GitLab", getGitLabAuthURL, "gitlab.com"},
		{"Gitee", getGiteeAuthURL, "gitee.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.fn(cfg, "test-state")
			if !contains(url, tt.contains) {
				t.Errorf("URL does not contain %s: %s", tt.contains, url)
			}
			if !contains(url, "client_id=test-client-id") {
				t.Errorf("URL does not contain client_id: %s", url)
			}
			if !contains(url, "state=test-state") {
				t.Errorf("URL does not contain state: %s", url)
			}
		})
	}
}

func TestHandleOAuthCallback(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
		OAuthProviders: map[string]OAuthConfig{
			"github": {
				ClientID:     "github-client-id",
				ClientSecret: "github-secret",
				RedirectURL:  "https://example.com/callback/github",
			},
		},
	}
	svc := NewService(cfg, nil)
	ctx := context.Background()

	t.Run("unsupported provider", func(t *testing.T) {
		_, _, _, err := svc.HandleOAuthCallback(ctx, "unsupported", "code", "state")
		if err == nil {
			t.Error("Expected error for unsupported provider")
		}
	})

	// Note: Testing actual OAuth callbacks would require mocking HTTP clients
	// The callback handlers return "not implemented" errors currently
	t.Run("github callback not implemented", func(t *testing.T) {
		_, _, _, err := svc.HandleOAuthCallback(ctx, "github", "code", "state")
		if err == nil {
			t.Error("Expected error for unimplemented callback")
		}
	})

	t.Run("google callback not implemented", func(t *testing.T) {
		cfg.OAuthProviders["google"] = OAuthConfig{ClientID: "test"}
		_, _, _, err := svc.HandleOAuthCallback(ctx, "google", "code", "state")
		if err == nil {
			t.Error("Expected error for unimplemented callback")
		}
	})

	t.Run("gitlab callback not implemented", func(t *testing.T) {
		cfg.OAuthProviders["gitlab"] = OAuthConfig{ClientID: "test"}
		_, _, _, err := svc.HandleOAuthCallback(ctx, "gitlab", "code", "state")
		if err == nil {
			t.Error("Expected error for unimplemented callback")
		}
	})

	t.Run("gitee callback not implemented", func(t *testing.T) {
		cfg.OAuthProviders["gitee"] = OAuthConfig{ClientID: "test"}
		_, _, _, err := svc.HandleOAuthCallback(ctx, "gitee", "code", "state")
		if err == nil {
			t.Error("Expected error for unimplemented callback")
		}
	})
}

func TestRevokeToken(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
	}
	svc := NewService(cfg, nil)
	ctx := context.Background()

	// RevokeToken currently does nothing but should not return an error
	err := svc.RevokeToken(ctx, "some-token")
	if err != nil {
		t.Errorf("RevokeToken returned error: %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "test-secret",
		JWTExpiration:     time.Hour,
		RefreshExpiration: 24 * time.Hour,
		Issuer:            "test-issuer",
	}
	svc := NewService(cfg, nil)
	ctx := context.Background()

	// RefreshToken without Redis returns ErrInvalidRefreshToken
	_, err := svc.RefreshToken(ctx, "some-refresh-token")
	if err == nil {
		t.Error("Expected error from RefreshToken")
	}
	if err != ErrInvalidRefreshToken {
		t.Errorf("Expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestErrors(t *testing.T) {
	errors := []struct {
		err      error
		expected string
	}{
		{ErrInvalidToken, "invalid token"},
		{ErrTokenExpired, "token expired"},
		{ErrRefreshExpired, "refresh token expired"},
		{ErrInvalidOAuthCode, "invalid OAuth code"},
		{ErrInvalidCredentials, "invalid credentials"},
		{ErrUserDisabled, "user is disabled"},
		{ErrEmailExists, "email already exists"},
		{ErrUsernameExists, "username already exists"},
		{ErrInvalidState, "invalid OAuth state"},
	}

	for _, tt := range errors {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}

func TestClaims(t *testing.T) {
	claims := &Claims{
		UserID:         1,
		Email:          "test@example.com",
		Username:       "testuser",
		OrganizationID: 123,
		Role:           "admin",
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %d, want 1", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", claims.Email)
	}
}

func TestTokenPair(t *testing.T) {
	pair := &TokenPair{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		TokenType:    "Bearer",
	}

	if pair.AccessToken != "access-token" {
		t.Errorf("AccessToken = %s, want access-token", pair.AccessToken)
	}
	if pair.TokenType != "Bearer" {
		t.Errorf("TokenType = %s, want Bearer", pair.TokenType)
	}
}

func TestConfig(t *testing.T) {
	cfg := &Config{
		JWTSecret:         "secret",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24,
		Issuer:            "issuer",
		OAuthProviders: map[string]OAuthConfig{
			"github": {
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://example.com/callback",
				Scopes:       []string{"user:email"},
			},
		},
	}

	if cfg.JWTSecret != "secret" {
		t.Errorf("JWTSecret = %s, want secret", cfg.JWTSecret)
	}
	if len(cfg.OAuthProviders) != 1 {
		t.Errorf("OAuthProviders count = %d, want 1", len(cfg.OAuthProviders))
	}
}

func TestOAuthConfig(t *testing.T) {
	cfg := OAuthConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://example.com/callback",
		Scopes:       []string{"user:email", "read:user"},
	}

	if cfg.ClientID != "client-id" {
		t.Errorf("ClientID = %s, want client-id", cfg.ClientID)
	}
	if len(cfg.Scopes) != 2 {
		t.Errorf("Scopes count = %d, want 2", len(cfg.Scopes))
	}
}

func TestRegisterRequest(t *testing.T) {
	req := &RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
		Name:     "Test User",
	}

	if req.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", req.Email)
	}
}

func TestLoginResult(t *testing.T) {
	result := &LoginResult{
		User: &user.User{
			ID:    1,
			Email: "test@example.com",
		},
		Token:        "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}

	if result.User.ID != 1 {
		t.Errorf("User.ID = %d, want 1", result.User.ID)
	}
	if result.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", result.ExpiresIn)
	}
}

func TestOAuthLoginRequest(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	req := &OAuthLoginRequest{
		Provider:       "github",
		ProviderUserID: "12345",
		Email:          "test@example.com",
		Username:       "testuser",
		Name:           "Test User",
		AvatarURL:      "https://example.com/avatar.png",
		AccessToken:    "access-token",
		RefreshToken:   "refresh-token",
		ExpiresAt:      &expiresAt,
	}

	if req.Provider != "github" {
		t.Errorf("Provider = %s, want github", req.Provider)
	}
	if req.ProviderUserID != "12345" {
		t.Errorf("ProviderUserID = %s, want 12345", req.ProviderUserID)
	}
}

func TestOAuthUserInfo(t *testing.T) {
	info := &OAuthUserInfo{
		ID:        "12345",
		Username:  "testuser",
		Email:     "test@example.com",
		Name:      "Test User",
		AvatarURL: "https://example.com/avatar.png",
	}

	if info.ID != "12345" {
		t.Errorf("ID = %s, want 12345", info.ID)
	}
	if info.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", info.Username)
	}
}

// ============================================================================
// Tests with real userService (using SQLite)
// ============================================================================

func TestLoginWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, userSvc)

	// Create a user first
	_, err := userSvc.Create(ctx, &userService.CreateRequest{
		Email:    "login@example.com",
		Username: "loginuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("successful login", func(t *testing.T) {
		result, err := svc.Login(ctx, "login@example.com", "password123")
		if err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
		if result.Token == "" {
			t.Error("Token should not be empty")
		}
		if result.RefreshToken == "" {
			t.Error("RefreshToken should not be empty")
		}
		if result.ExpiresIn != int64(time.Hour.Seconds()) {
			t.Errorf("ExpiresIn = %d, want %d", result.ExpiresIn, int64(time.Hour.Seconds()))
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		_, err := svc.Login(ctx, "login@example.com", "wrongpassword")
		if err == nil {
			t.Error("Expected error for invalid credentials")
		}
		if err != ErrInvalidCredentials {
			t.Errorf("Expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.Login(ctx, "nonexistent@example.com", "password")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
		if err != ErrInvalidCredentials {
			t.Errorf("Expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("disabled user", func(t *testing.T) {
		// Create and deactivate a user
		u, _ := userSvc.Create(ctx, &userService.CreateRequest{
			Email:    "disabled@example.com",
			Username: "disableduser",
			Password: "password123",
		})
		db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", u.ID)

		_, err := svc.Login(ctx, "disabled@example.com", "password123")
		if err == nil {
			t.Error("Expected error for disabled user")
		}
		if err != ErrUserDisabled {
			t.Errorf("Expected ErrUserDisabled, got %v", err)
		}
	})
}

func TestRegisterWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, userSvc)

	t.Run("successful registration", func(t *testing.T) {
		req := &RegisterRequest{
			Email:    "newuser@example.com",
			Username: "newuser",
			Password: "password123",
			Name:     "New User",
		}

		result, err := svc.Register(ctx, req)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
		if result.User.Email != "newuser@example.com" {
			t.Errorf("Email = %s, want newuser@example.com", result.User.Email)
		}
		if result.Token == "" {
			t.Error("Token should not be empty")
		}
		if result.RefreshToken == "" {
			t.Error("RefreshToken should not be empty")
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		// First registration
		svc.Register(ctx, &RegisterRequest{
			Email:    "dupe@example.com",
			Username: "dupeuser1",
			Password: "password123",
		})

		// Second registration with same email
		_, err := svc.Register(ctx, &RegisterRequest{
			Email:    "dupe@example.com",
			Username: "dupeuser2",
			Password: "password123",
		})
		if err == nil {
			t.Error("Expected error for duplicate email")
		}
		if err != ErrEmailExists {
			t.Errorf("Expected ErrEmailExists, got %v", err)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		// First registration
		svc.Register(ctx, &RegisterRequest{
			Email:    "unique1@example.com",
			Username: "sameusername",
			Password: "password123",
		})

		// Second registration with same username
		_, err := svc.Register(ctx, &RegisterRequest{
			Email:    "unique2@example.com",
			Username: "sameusername",
			Password: "password123",
		})
		if err == nil {
			t.Error("Expected error for duplicate username")
		}
		if err != ErrUsernameExists {
			t.Errorf("Expected ErrUsernameExists, got %v", err)
		}
	})
}

func TestRefreshTokensWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, userSvc)

	// Create a user first
	u, err := userSvc.Create(ctx, &userService.CreateRequest{
		Email:    "refresh@example.com",
		Username: "refreshuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("successful refresh", func(t *testing.T) {
		// Generate initial tokens
		tokens, err := svc.GenerateTokenPair(u, 100, "member")
		if err != nil {
			t.Fatalf("GenerateTokenPair failed: %v", err)
		}

		// Refresh tokens
		newTokens, err := svc.RefreshTokens(ctx, tokens.AccessToken, tokens.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshTokens failed: %v", err)
		}

		if newTokens.AccessToken == "" {
			t.Error("New AccessToken should not be empty")
		}
		if newTokens.RefreshToken == "" {
			t.Error("New RefreshToken should not be empty")
		}

		// Validate the new access token has the same claims
		claims, err := svc.ValidateToken(newTokens.AccessToken)
		if err != nil {
			t.Fatalf("ValidateToken failed: %v", err)
		}
		if claims.UserID != u.ID {
			t.Errorf("UserID = %d, want %d", claims.UserID, u.ID)
		}
		if claims.OrganizationID != 100 {
			t.Errorf("OrganizationID = %d, want 100", claims.OrganizationID)
		}
		if claims.Role != "member" {
			t.Errorf("Role = %s, want member", claims.Role)
		}
	})

	t.Run("refresh with expired token", func(t *testing.T) {
		// Generate an expired token
		expiredCfg := &Config{
			JWTSecret:     "test-secret-key-at-least-32-bytes",
			JWTExpiration: -time.Hour, // Already expired
			Issuer:        "test-issuer",
		}
		expiredSvc := NewService(expiredCfg, userSvc)
		expiredTokens, _ := expiredSvc.GenerateTokenPair(u, 50, "viewer")

		// RefreshTokens should still work with expired access token
		newTokens, err := svc.RefreshTokens(ctx, expiredTokens.AccessToken, expiredTokens.RefreshToken)
		if err != nil {
			t.Fatalf("RefreshTokens with expired access token failed: %v", err)
		}
		if newTokens.AccessToken == "" {
			t.Error("New AccessToken should not be empty")
		}
	})

	t.Run("user not found for refresh", func(t *testing.T) {
		// Create token for a non-existent user
		nonExistentUser := &user.User{
			ID:       99999,
			Email:    "nonexistent@example.com",
			Username: "nonexistent",
		}
		tokens, _ := svc.GenerateTokenPair(nonExistentUser, 0, "")

		_, err := svc.RefreshTokens(ctx, tokens.AccessToken, tokens.RefreshToken)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestOAuthLoginWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}

	svc := NewService(cfg, userSvc)

	t.Run("creates new user via oauth", func(t *testing.T) {
		req := &OAuthLoginRequest{
			Provider:       "github",
			ProviderUserID: "gh_12345",
			Email:          "oauthuser@example.com",
			Username:       "oauthuser",
			Name:           "OAuth User",
			AvatarURL:      "https://github.com/avatar.png",
			AccessToken:    "gh_access_token",
			RefreshToken:   "gh_refresh_token",
		}

		result, err := svc.OAuthLogin(ctx, req)
		if err != nil {
			t.Fatalf("OAuthLogin failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
		if result.User.Email != "oauthuser@example.com" {
			t.Errorf("Email = %s, want oauthuser@example.com", result.User.Email)
		}
		if result.Token == "" {
			t.Error("Token should not be empty")
		}
	})

	t.Run("returns existing user via oauth", func(t *testing.T) {
		// First OAuth login creates user
		req := &OAuthLoginRequest{
			Provider:       "github",
			ProviderUserID: "gh_99999",
			Email:          "existingoauth@example.com",
			Username:       "existingoauth",
			Name:           "Existing OAuth User",
		}
		result1, _ := svc.OAuthLogin(ctx, req)

		// Second OAuth login returns same user
		result2, err := svc.OAuthLogin(ctx, req)
		if err != nil {
			t.Fatalf("Second OAuthLogin failed: %v", err)
		}
		if result2.User.ID != result1.User.ID {
			t.Errorf("User ID mismatch: %d != %d", result2.User.ID, result1.User.ID)
		}
	})

	t.Run("oauth without access token", func(t *testing.T) {
		req := &OAuthLoginRequest{
			Provider:       "gitlab",
			ProviderUserID: "gl_12345",
			Email:          "notoken@example.com",
			Username:       "notoken",
			Name:           "No Token User",
			// No AccessToken
		}

		result, err := svc.OAuthLogin(ctx, req)
		if err != nil {
			t.Fatalf("OAuthLogin failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
	})

	t.Run("oauth with token expiration", func(t *testing.T) {
		expiresAt := time.Now().Add(time.Hour)
		req := &OAuthLoginRequest{
			Provider:       "google",
			ProviderUserID: "google_12345",
			Email:          "googleuser@example.com",
			Username:       "googleuser",
			Name:           "Google User",
			AccessToken:    "google_access_token",
			RefreshToken:   "google_refresh_token",
			ExpiresAt:      &expiresAt,
		}

		result, err := svc.OAuthLogin(ctx, req)
		if err != nil {
			t.Fatalf("OAuthLogin failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
	})
}

// Additional edge case tests

func TestHandleOAuthCallbackWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
		OAuthProviders: map[string]OAuthConfig{
			"github": {
				ClientID:     "github-client-id",
				ClientSecret: "github-secret",
				RedirectURL:  "https://example.com/callback/github",
			},
			"google": {
				ClientID:     "google-client-id",
				ClientSecret: "google-secret",
				RedirectURL:  "https://example.com/callback/google",
			},
			"gitlab": {
				ClientID:     "gitlab-client-id",
				ClientSecret: "gitlab-secret",
				RedirectURL:  "https://example.com/callback/gitlab",
			},
			"gitee": {
				ClientID:     "gitee-client-id",
				ClientSecret: "gitee-secret",
				RedirectURL:  "https://example.com/callback/gitee",
			},
		},
	}
	svc := NewService(cfg, userSvc)
	ctx := context.Background()

	// Test default case in HandleOAuthCallback - unsupported provider after cfg check
	t.Run("default switch case", func(t *testing.T) {
		// Add a fake provider to config to pass the first check
		cfg.OAuthProviders["fake"] = OAuthConfig{ClientID: "fake"}
		_, _, _, err := svc.HandleOAuthCallback(ctx, "fake", "code", "state")
		if err == nil {
			t.Error("Expected error for fake provider")
		}
		delete(cfg.OAuthProviders, "fake")
	})
}

func TestGetOAuthURLDefault(t *testing.T) {
	cfg := &Config{
		JWTSecret:     "test-secret",
		JWTExpiration: time.Hour,
		Issuer:        "test-issuer",
		OAuthProviders: map[string]OAuthConfig{
			"customauth": {
				ClientID: "custom-id",
			},
		},
	}
	svc := NewService(cfg, nil)

	// Test default case in GetOAuthURL
	t.Run("default switch case", func(t *testing.T) {
		_, err := svc.GetOAuthURL("customauth", "state")
		if err == nil {
			t.Error("Expected error for unsupported provider in switch")
		}
	})
}

func TestLoginErrors(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}
	svc := NewService(cfg, userSvc)

	// Create a user first
	_, err := userSvc.Create(ctx, &userService.CreateRequest{
		Email:    "errortest@example.com",
		Username: "errortest",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("generic db error", func(t *testing.T) {
		// This tests that unexpected errors pass through
		// We can't easily simulate a DB error, so we just ensure other errors work
		_, err := svc.Login(ctx, "errortest@example.com", "wrongpassword")
		if err != ErrInvalidCredentials {
			t.Errorf("Expected ErrInvalidCredentials for wrong password, got %v", err)
		}
	})
}

func TestRegisterErrors(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}
	svc := NewService(cfg, userSvc)

	t.Run("registration without name", func(t *testing.T) {
		req := &RegisterRequest{
			Email:    "noname@example.com",
			Username: "noname",
			Password: "password123",
			// Name is empty
		}
		result, err := svc.Register(ctx, req)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}
		if result.User.Name != nil {
			t.Error("Expected nil Name")
		}
	})
}

func TestOAuthLoginErrors(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(db)
	ctx := context.Background()

	cfg := &Config{
		JWTSecret:         "test-secret-key-at-least-32-bytes",
		JWTExpiration:     time.Hour,
		RefreshExpiration: time.Hour * 24 * 7,
		Issuer:            "test-issuer",
	}
	svc := NewService(cfg, userSvc)

	t.Run("oauth without any optional fields", func(t *testing.T) {
		req := &OAuthLoginRequest{
			Provider:       "github",
			ProviderUserID: "minimal_12345",
			Email:          "minimal@example.com",
			Username:       "minimal",
			// All optional fields omitted
		}
		result, err := svc.OAuthLogin(ctx, req)
		if err != nil {
			t.Fatalf("OAuthLogin failed: %v", err)
		}
		if result.User == nil {
			t.Error("User should not be nil")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
