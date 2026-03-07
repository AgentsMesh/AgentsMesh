package auth

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

func TestRegisterWithUserService(t *testing.T) {
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

func TestRegisterErrors(t *testing.T) {
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
