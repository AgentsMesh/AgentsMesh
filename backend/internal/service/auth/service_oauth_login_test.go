package auth

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

func TestOAuthLoginWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(infra.NewUserRepository(db))
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

func TestOAuthLoginErrors(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(infra.NewUserRepository(db))
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
