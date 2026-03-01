package auth

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

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
