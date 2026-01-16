package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Registration Token Tests ---

func TestCreateRegistrationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	token, err := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	if err != nil {
		t.Fatalf("failed to create registration token: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Verify the token was stored
	var regToken runner.RegistrationToken
	if err := db.First(&regToken).Error; err != nil {
		t.Fatalf("failed to find registration token: %v", err)
	}
	if regToken.OrganizationID != 1 {
		t.Errorf("expected OrganizationID 1, got %d", regToken.OrganizationID)
	}
	if regToken.Description != "Test Token" {
		t.Errorf("expected Description 'Test Token', got %s", regToken.Description)
	}
	if !regToken.IsActive {
		t.Error("expected token to be active")
	}
}

func TestListRegistrationTokens(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	// Create multiple tokens
	service.CreateRegistrationToken(ctx, 1, 1, "Token 1", nil, nil)
	service.CreateRegistrationToken(ctx, 1, 1, "Token 2", nil, nil)
	service.CreateRegistrationToken(ctx, 2, 1, "Token for Org 2", nil, nil)

	tokens, err := service.ListRegistrationTokens(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list registration tokens: %v", err)
	}

	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for org 1, got %d", len(tokens))
	}
}

func TestRevokeRegistrationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)

	// Get the token ID
	var token runner.RegistrationToken
	db.First(&token)

	err := service.RevokeRegistrationToken(ctx, token.ID)
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Token should be invalid now
	_, err = service.ValidateRegistrationToken(ctx, plain)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken after revoke, got %v", err)
	}
}
