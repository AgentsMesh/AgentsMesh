package user

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func TestAuthenticate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create a user with password
	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}
	service.Create(ctx, req)

	// Authenticate
	user, err := service.Authenticate(ctx, "test@example.com", "password123")
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected Email 'test@example.com', got %s", user.Email)
	}
}

func TestAuthenticateInvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
	}
	service.Create(ctx, req)

	_, err := service.Authenticate(ctx, "test@example.com", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateUserNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	_, err := service.Authenticate(ctx, "nonexistent@example.com", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateNoPassword(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	// Create user without password
	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
	}
	service.Create(ctx, req)

	_, err := service.Authenticate(ctx, "test@example.com", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateInactiveUser(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "inactive@example.com",
		Username: "inactiveuser",
		Password: "password123",
	}
	created, _ := service.Create(ctx, req)

	// Deactivate user
	db.Exec("UPDATE users SET is_active = 0 WHERE id = ?", created.ID)

	_, err := service.Authenticate(ctx, "inactive@example.com", "password123")
	if err != ErrUserInactive {
		t.Errorf("expected ErrUserInactive, got %v", err)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	req := &CreateRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "oldpassword",
	}
	created, _ := service.Create(ctx, req)

	// Update password
	err := service.UpdatePassword(ctx, created.ID, "newpassword")
	if err != nil {
		t.Fatalf("failed to update password: %v", err)
	}

	// Should be able to authenticate with new password
	_, err = service.Authenticate(ctx, "test@example.com", "newpassword")
	if err != nil {
		t.Errorf("expected successful authentication, got %v", err)
	}
}
