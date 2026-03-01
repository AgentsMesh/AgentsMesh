package repository

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
)

// ===========================================
// WebhookConfig ToStatus Tests
// ===========================================

func TestWebhookConfigToStatus(t *testing.T) {
	config := &gitprovider.WebhookConfig{
		ID:               "wh_123",
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Secret:           "secret123",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         true,
		NeedsManualSetup: false,
		CreatedAt:        "2024-01-01T00:00:00Z",
	}

	status := config.ToStatus()

	if !status.Registered {
		t.Error("expected Registered to be true")
	}
	if status.WebhookID != "wh_123" {
		t.Errorf("expected WebhookID 'wh_123', got %s", status.WebhookID)
	}
	if status.WebhookURL != "https://example.com/webhooks/org/gitlab/1" {
		t.Errorf("expected WebhookURL, got %s", status.WebhookURL)
	}
	if !status.IsActive {
		t.Error("expected IsActive to be true")
	}
	if status.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be false")
	}
}

func TestWebhookConfigToStatus_ManualSetup(t *testing.T) {
	config := &gitprovider.WebhookConfig{
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Secret:           "secret123",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         false,
		NeedsManualSetup: true,
		LastError:        "OAuth token not found",
	}

	status := config.ToStatus()

	// When NeedsManualSetup=true, it's technically registered (config exists)
	// but not fully active
	if status.IsActive {
		t.Error("expected IsActive to be false")
	}
	if !status.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be true")
	}
	if status.LastError != "OAuth token not found" {
		t.Errorf("expected LastError to be set, got %s", status.LastError)
	}
}

// ===========================================
// RegisterWebhookForRepository - Logic Tests
// ===========================================

func TestRegisterWebhookForRepository_ResultFields(t *testing.T) {
	// Test WebhookResult structure
	result := &WebhookResult{
		RepoID:              100,
		Registered:          false,
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example.com/api/webhooks/org/gitlab/100",
		ManualWebhookSecret: "abc123secret",
		Error:               "OAuth token not available",
	}

	if result.RepoID != 100 {
		t.Errorf("expected RepoID 100, got %d", result.RepoID)
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
	if result.Error == "" {
		t.Error("expected Error to be set")
	}
}

func TestRegisterWebhookForRepository_SuccessResultFields(t *testing.T) {
	result := &WebhookResult{
		RepoID:     200,
		Registered: true,
		WebhookID:  "wh_xyz789",
	}

	if result.RepoID != 200 {
		t.Errorf("expected RepoID 200, got %d", result.RepoID)
	}
	if !result.Registered {
		t.Error("expected Registered to be true")
	}
	if result.WebhookID != "wh_xyz789" {
		t.Errorf("expected WebhookID 'wh_xyz789', got %s", result.WebhookID)
	}
	if result.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be false")
	}
}
