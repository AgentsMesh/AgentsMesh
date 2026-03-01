package auth

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
)

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
