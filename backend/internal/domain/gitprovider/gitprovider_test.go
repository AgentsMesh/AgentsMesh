package gitprovider

import (
	"encoding/json"
	"testing"
)

// ===========================================
// WebhookConfig Tests
// ===========================================

func TestWebhookConfig_ToStatus_Nil(t *testing.T) {
	var wc *WebhookConfig
	status := wc.ToStatus()

	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if status.Registered {
		t.Error("expected Registered to be false for nil config")
	}
}

func TestWebhookConfig_ToStatus_AutoRegistered(t *testing.T) {
	wc := &WebhookConfig{
		ID:        "wh_123",
		URL:       "https://example.com/webhooks/org/gitlab/1",
		Secret:    "secret123",
		Events:    []string{"merge_request", "pipeline"},
		IsActive:  true,
		CreatedAt: "2026-02-06T10:00:00Z",
	}

	status := wc.ToStatus()

	if !status.Registered {
		t.Error("expected Registered to be true")
	}
	if status.WebhookID != "wh_123" {
		t.Errorf("unexpected WebhookID: %s", status.WebhookID)
	}
	if status.WebhookURL != "https://example.com/webhooks/org/gitlab/1" {
		t.Errorf("unexpected WebhookURL: %s", status.WebhookURL)
	}
	if len(status.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(status.Events))
	}
	if !status.IsActive {
		t.Error("expected IsActive to be true")
	}
	if status.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be false")
	}
	if status.RegisteredAt != "2026-02-06T10:00:00Z" {
		t.Errorf("unexpected RegisteredAt: %s", status.RegisteredAt)
	}
}

func TestWebhookConfig_ToStatus_ManualSetup(t *testing.T) {
	wc := &WebhookConfig{
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Secret:           "secret123",
		Events:           []string{"merge_request", "pipeline"},
		IsActive:         false,
		NeedsManualSetup: true,
		LastError:        "OAuth token not available",
	}

	status := wc.ToStatus()

	if !status.Registered {
		t.Error("expected Registered to be true when NeedsManualSetup is true")
	}
	if status.WebhookID != "" {
		t.Errorf("expected empty WebhookID, got %s", status.WebhookID)
	}
	if !status.NeedsManualSetup {
		t.Error("expected NeedsManualSetup to be true")
	}
	if status.IsActive {
		t.Error("expected IsActive to be false")
	}
	if status.LastError != "OAuth token not available" {
		t.Errorf("unexpected LastError: %s", status.LastError)
	}
}

func TestWebhookConfig_ToStatus_NoIDNoManualSetup(t *testing.T) {
	wc := &WebhookConfig{
		URL:              "https://example.com/webhooks/org/gitlab/1",
		Events:           []string{"merge_request"},
		IsActive:         false,
		NeedsManualSetup: false,
	}

	status := wc.ToStatus()

	if status.Registered {
		t.Error("expected Registered to be false without ID or NeedsManualSetup")
	}
}

func TestWebhookConfig_ToStatus_SecretNotExposed(t *testing.T) {
	wc := &WebhookConfig{
		ID:     "wh_123",
		URL:    "https://example.com/webhooks/org/gitlab/1",
		Secret: "super_secret_value",
	}

	status := wc.ToStatus()

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal status: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if _, hasSecret := parsed["secret"]; hasSecret {
		t.Error("WebhookStatus should not expose secret field")
	}
}

// ===========================================
// WebhookConfig JSON Serialization Tests
// ===========================================

func TestWebhookConfig_JSONSerialization(t *testing.T) {
	wc := &WebhookConfig{
		ID:               "wh_abc123",
		URL:              "https://example.com/webhooks/test",
		Secret:           "secret_value",
		Events:           []string{"merge_request", "pipeline", "push"},
		IsActive:         true,
		NeedsManualSetup: false,
		LastError:        "",
		CreatedAt:        "2026-02-06T10:00:00Z",
	}

	data, err := json.Marshal(wc)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed WebhookConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != wc.ID {
		t.Errorf("ID mismatch: %s vs %s", parsed.ID, wc.ID)
	}
	if parsed.URL != wc.URL {
		t.Errorf("URL mismatch: %s vs %s", parsed.URL, wc.URL)
	}
	if parsed.Secret != wc.Secret {
		t.Errorf("Secret mismatch: %s vs %s", parsed.Secret, wc.Secret)
	}
	if len(parsed.Events) != len(wc.Events) {
		t.Errorf("Events length mismatch: %d vs %d", len(parsed.Events), len(wc.Events))
	}
	if parsed.IsActive != wc.IsActive {
		t.Error("IsActive mismatch")
	}
	if parsed.NeedsManualSetup != wc.NeedsManualSetup {
		t.Error("NeedsManualSetup mismatch")
	}
}

func TestWebhookConfig_JSONOmitEmpty(t *testing.T) {
	wc := &WebhookConfig{
		ID:       "wh_123",
		URL:      "https://example.com/webhooks/test",
		Events:   []string{"merge_request"},
		IsActive: true,
	}

	data, err := json.Marshal(wc)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if _, has := parsed["secret"]; has {
		t.Error("expected secret to be omitted when empty")
	}
	if _, has := parsed["last_error"]; has {
		t.Error("expected last_error to be omitted when empty")
	}
	if _, has := parsed["created_at"]; has {
		t.Error("expected created_at to be omitted when empty")
	}
}

// ===========================================
// WebhookStatus JSON Serialization Tests
// ===========================================

func TestWebhookStatus_JSONSerialization(t *testing.T) {
	status := &WebhookStatus{
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

	var parsed WebhookStatus
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Registered != status.Registered {
		t.Error("Registered mismatch")
	}
	if parsed.WebhookID != status.WebhookID {
		t.Errorf("WebhookID mismatch: %s vs %s", parsed.WebhookID, status.WebhookID)
	}
	if parsed.IsActive != status.IsActive {
		t.Error("IsActive mismatch")
	}
}

func TestWebhookStatus_JSONOmitEmpty(t *testing.T) {
	status := &WebhookStatus{
		Registered: false,
		IsActive:   false,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if _, has := parsed["webhook_id"]; has {
		t.Error("expected webhook_id to be omitted when empty")
	}
	if _, has := parsed["webhook_url"]; has {
		t.Error("expected webhook_url to be omitted when empty")
	}
	if _, has := parsed["events"]; has {
		t.Error("expected events to be omitted when empty/nil")
	}
	if _, has := parsed["last_error"]; has {
		t.Error("expected last_error to be omitted when empty")
	}
	if _, has := parsed["registered_at"]; has {
		t.Error("expected registered_at to be omitted when empty")
	}
}
