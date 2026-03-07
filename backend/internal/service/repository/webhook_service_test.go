package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

// ===========================================
// WebhookService Constructor Tests
// ===========================================

func TestNewWebhookService(t *testing.T) {
	db := setupWebhookTestDB(t)
	cfg := &config.Config{}
	repo := infra.NewGitProviderRepository(db)
	svc := NewWebhookService(repo, cfg, nil, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Error("repo not set correctly")
	}
	if svc.cfg != cfg {
		t.Error("config not set correctly")
	}
}

// ===========================================
// WebhookResult Tests
// ===========================================

func TestWebhookResult_Fields(t *testing.T) {
	result := &WebhookResult{
		RepoID:              123,
		Registered:          true,
		WebhookID:           "wh_abc123",
		NeedsManualSetup:    false,
		ManualWebhookURL:    "",
		ManualWebhookSecret: "",
		Error:               "",
	}

	if result.RepoID != 123 {
		t.Errorf("expected RepoID 123, got %d", result.RepoID)
	}
	if !result.Registered {
		t.Error("expected Registered to be true")
	}
	if result.WebhookID != "wh_abc123" {
		t.Errorf("expected WebhookID 'wh_abc123', got %s", result.WebhookID)
	}
}

func TestWebhookResult_ManualSetup(t *testing.T) {
	result := &WebhookResult{
		RepoID:              456,
		Registered:          false,
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example.com/webhooks/org/gitlab/456",
		ManualWebhookSecret: "secret123",
		Error:               "no OAuth token",
	}

	if result.Registered {
		t.Error("expected Registered to be false")
	}
	if !result.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be true")
	}
	if result.ManualWebhookURL == "" {
		t.Error("expected ManualWebhookURL to be set")
	}
	if result.ManualWebhookSecret == "" {
		t.Error("expected ManualWebhookSecret to be set")
	}
}

// ===========================================
// Error Variables Tests
// ===========================================

func TestWebhookErrorVariables(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrNoAccessToken, "no access token available for git provider"},
		{ErrWebhookNotFound, "webhook not found"},
		{ErrWebhookExists, "webhook already registered"},
		{ErrProviderMismatch, "user provider type does not match repository"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("expected error message %q, got %q", tt.expected, tt.err.Error())
		}
	}
}
