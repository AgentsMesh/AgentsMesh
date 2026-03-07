package auth

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

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

func TestHandleOAuthCallbackWithUserService(t *testing.T) {
	db := setupTestDB(t)
	userSvc := userService.NewService(infra.NewUserRepository(db))

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
