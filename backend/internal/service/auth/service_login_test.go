package auth

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

func TestLoginWithUserService(t *testing.T) {
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

func TestLoginErrors(t *testing.T) {
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
