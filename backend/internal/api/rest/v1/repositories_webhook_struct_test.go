package v1

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/repository"
)

// ===========================================
// WebhookResult Structure Tests
// ===========================================

func TestWebhookResult_Serialization(t *testing.T) {
	result := &repository.WebhookResult{
		RepoID:              123,
		Registered:          true,
		WebhookID:           "wh_abc123",
		NeedsManualSetup:    false,
		ManualWebhookURL:    "",
		ManualWebhookSecret: "",
		Error:               "",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal WebhookResult: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["repo_id"].(float64) != 123 {
		t.Errorf("unexpected repo_id: %v", parsed["repo_id"])
	}
	if parsed["registered"] != true {
		t.Error("expected registered to be true")
	}
	if parsed["webhook_id"] != "wh_abc123" {
		t.Errorf("unexpected webhook_id: %v", parsed["webhook_id"])
	}
}

func TestWebhookResult_ManualSetupSerialization(t *testing.T) {
	result := &repository.WebhookResult{
		RepoID:              456,
		Registered:          false,
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example.com/webhooks/org/gitlab/456",
		ManualWebhookSecret: "secret123",
		Error:               "OAuth token not available",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["needs_manual_setup"] != true {
		t.Error("expected needs_manual_setup to be true")
	}
	if parsed["manual_webhook_url"] != "https://example.com/webhooks/org/gitlab/456" {
		t.Errorf("unexpected manual_webhook_url: %v", parsed["manual_webhook_url"])
	}
}

// ===========================================
// WebhookStatus Structure Tests
// ===========================================

func TestWebhookStatus_Serialization(t *testing.T) {
	status := &gitprovider.WebhookStatus{
		Registered:       true,
		WebhookID:        "wh_test",
		WebhookURL:       "https://example.com/webhooks/test",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         true,
		NeedsManualSetup: false,
		LastError:        "",
		RegisteredAt:     "2026-02-06T10:00:00Z",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["registered"] != true {
		t.Error("expected registered to be true")
	}
	if parsed["is_active"] != true {
		t.Error("expected is_active to be true")
	}

	events := parsed["events"].([]interface{})
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

// ===========================================
// Repository.WebhookConfig Tests
// ===========================================

func TestWebhookConfig_Fields(t *testing.T) {
	config := &gitprovider.WebhookConfig{
		ID:               "wh_123",
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Secret:           "secret123",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         true,
		NeedsManualSetup: false,
		LastError:        "",
		CreatedAt:        "2026-02-06T10:00:00Z",
	}

	if config.ID != "wh_123" {
		t.Errorf("unexpected ID: %s", config.ID)
	}
	if config.Secret != "secret123" {
		t.Errorf("unexpected Secret: %s", config.Secret)
	}
	if len(config.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(config.Events))
	}
	if !config.IsActive {
		t.Error("expected IsActive to be true")
	}
}

func TestWebhookConfig_ManualSetupMode(t *testing.T) {
	config := &gitprovider.WebhookConfig{
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Secret:           "manual_secret",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         false,
		NeedsManualSetup: true,
		LastError:        "OAuth token not found",
	}

	if !config.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be true")
	}
	if config.IsActive {
		t.Error("expected IsActive to be false for manual setup")
	}
	if config.LastError != "OAuth token not found" {
		t.Errorf("unexpected LastError: %s", config.LastError)
	}
}

// ===========================================
// Error Handling Tests
// ===========================================

func TestRepositoryWebhookErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrWebhookNotFound",
			err:      repository.ErrWebhookNotFound,
			expected: "webhook not found",
		},
		{
			name:     "ErrWebhookExists",
			err:      repository.ErrWebhookExists,
			expected: "webhook already registered",
		},
		{
			name:     "ErrNoAccessToken",
			err:      repository.ErrNoAccessToken,
			expected: "no access token available for git provider",
		},
		{
			name:     "ErrProviderMismatch",
			err:      repository.ErrProviderMismatch,
			expected: "user provider type does not match repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	// Test that errors can be compared with errors.Is
	err := repository.ErrWebhookNotFound
	if !errors.Is(err, repository.ErrWebhookNotFound) {
		t.Error("errors.Is should match ErrWebhookNotFound")
	}

	err = repository.ErrRepositoryNotFound
	if !errors.Is(err, repository.ErrRepositoryNotFound) {
		t.Error("errors.Is should match ErrRepositoryNotFound")
	}
}
