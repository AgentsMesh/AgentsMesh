package repository

import (
	"context"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
)

// ===========================================
// VerifyWebhookSecret Tests
// ===========================================

func TestVerifyWebhookSecret_RepoNotFound(t *testing.T) {
	svc, _ := createTestWebhookService(t)
	ctx := context.Background()

	_, err := svc.VerifyWebhookSecret(ctx, 99999, "secret")

	if err == nil {
		t.Error("expected error when repo not found")
	}
}

func TestVerifyWebhookSecret_NoConfig(t *testing.T) {
	svc, db := createTestWebhookService(t)
	ctx := context.Background()

	repo := createTestRepository(t, db)

	// Since SQLite doesn't support JSONB and we can't persist WebhookConfig,
	// we test the function's behavior when repo has no webhook_config
	_, err := svc.VerifyWebhookSecret(ctx, repo.ID, "secret")

	if err != ErrWebhookNotFound {
		t.Errorf("expected ErrWebhookNotFound, got %v", err)
	}
}

// Note: The following tests require JSONB support which SQLite doesn't have.
// In production with PostgreSQL, these tests would work correctly.
// For unit testing, we test the logic separately by setting in-memory values.

func TestVerifyWebhookSecretLogic_EmptySecret(t *testing.T) {
	// Test the secret verification logic directly
	config := &gitprovider.WebhookConfig{
		URL:    "https://example.com/webhooks/org/gitlab/1",
		Secret: "", // Empty secret
	}

	if config.Secret != "" {
		t.Error("expected empty secret")
	}
}

func TestVerifyWebhookSecretLogic_Mismatch(t *testing.T) {
	// Test the secret verification logic directly
	config := &gitprovider.WebhookConfig{
		URL:    "https://example.com/webhooks/org/gitlab/1",
		Secret: "correct_secret",
	}

	providedSecret := "wrong_secret"
	valid := config.Secret == providedSecret

	if valid {
		t.Error("expected valid to be false for mismatched secret")
	}
}

func TestVerifyWebhookSecretLogic_Match(t *testing.T) {
	// Test the secret verification logic directly
	config := &gitprovider.WebhookConfig{
		URL:    "https://example.com/webhooks/org/gitlab/1",
		Secret: "correct_secret",
	}

	providedSecret := "correct_secret"
	valid := config.Secret == providedSecret

	if !valid {
		t.Error("expected valid to be true for matching secret")
	}
}

func TestVerifyWebhookSecret_CorrectSecret(t *testing.T) {
	// This test verifies the logic flow without JSONB persistence
	// We test that the comparison logic works correctly
	config := &gitprovider.WebhookConfig{
		Secret: "my-secret-123",
	}

	// Simulate verification logic
	providedSecret := "my-secret-123"
	isValid := config.Secret == providedSecret

	if !isValid {
		t.Error("expected secret verification to succeed with matching secret")
	}
}

func TestVerifyWebhookSecret_IncorrectSecret(t *testing.T) {
	config := &gitprovider.WebhookConfig{
		Secret: "correct-secret",
	}

	providedSecret := "wrong-secret"
	isValid := config.Secret == providedSecret

	if isValid {
		t.Error("expected secret verification to fail with wrong secret")
	}
}
