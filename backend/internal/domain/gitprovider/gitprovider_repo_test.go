package gitprovider

import (
	"encoding/json"
	"testing"
	"time"
)

// ===========================================
// Repository Tests
// ===========================================

func TestRepository_TableName(t *testing.T) {
	repo := Repository{}
	if repo.TableName() != "repositories" {
		t.Errorf("unexpected table name: %s", repo.TableName())
	}
}

func TestRepository_Fields(t *testing.T) {
	now := time.Now()
	ticketPrefix := "PROJ"
	userID := int64(123)

	repo := &Repository{
		ID:               1,
		OrganizationID:   100,
		ProviderType:     ProviderTypeGitLab,
		ProviderBaseURL:  "https://gitlab.com",
		HttpCloneURL:     "https://gitlab.com/org/repo.git",
		ExternalID:       "12345",
		Name:             "test-repo",
		Slug:         "org/test-repo",
		DefaultBranch:    "main",
		TicketPrefix:     &ticketPrefix,
		Visibility:       "organization",
		ImportedByUserID: &userID,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if repo.ID != 1 {
		t.Errorf("unexpected ID: %d", repo.ID)
	}
	if repo.OrganizationID != 100 {
		t.Errorf("unexpected OrganizationID: %d", repo.OrganizationID)
	}
	if repo.ProviderType != "gitlab" {
		t.Errorf("unexpected ProviderType: %s", repo.ProviderType)
	}
	if repo.ProviderBaseURL != "https://gitlab.com" {
		t.Errorf("unexpected ProviderBaseURL: %s", repo.ProviderBaseURL)
	}
	if repo.TicketPrefix == nil || *repo.TicketPrefix != "PROJ" {
		t.Error("unexpected TicketPrefix")
	}
	if repo.ImportedByUserID == nil || *repo.ImportedByUserID != 123 {
		t.Error("unexpected ImportedByUserID")
	}
}

func TestRepository_WithWebhookConfig(t *testing.T) {
	repo := &Repository{
		ID:              1,
		OrganizationID:  100,
		ProviderType:    ProviderTypeGitHub,
		ProviderBaseURL: "https://github.com",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		WebhookConfig: &WebhookConfig{
			ID:       "wh_123",
			URL:      "https://example.com/webhooks/org/github/1",
			Secret:   "secret",
			Events:   []string{"pull_request", "push"},
			IsActive: true,
		},
	}

	if repo.WebhookConfig == nil {
		t.Fatal("expected WebhookConfig to be set")
	}
	if repo.WebhookConfig.ID != "wh_123" {
		t.Errorf("unexpected WebhookConfig.ID: %s", repo.WebhookConfig.ID)
	}
	if len(repo.WebhookConfig.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(repo.WebhookConfig.Events))
	}
}

func TestRepository_JSONSerialization(t *testing.T) {
	ticketPrefix := "TEST"
	repo := &Repository{
		ID:              1,
		OrganizationID:  100,
		ProviderType:    ProviderTypeGitLab,
		ProviderBaseURL: "https://gitlab.com",
		HttpCloneURL:    "https://gitlab.com/org/repo.git",
		ExternalID:      "12345",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		DefaultBranch:   "main",
		TicketPrefix:    &ticketPrefix,
		Visibility:      "organization",
		IsActive:        true,
		WebhookConfig: &WebhookConfig{
			ID:       "wh_123",
			URL:      "https://example.com/webhooks",
			Events:   []string{"merge_request"},
			IsActive: true,
		},
	}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["id"].(float64) != 1 {
		t.Errorf("unexpected id: %v", parsed["id"])
	}
	if parsed["provider_type"] != "gitlab" {
		t.Errorf("unexpected provider_type: %v", parsed["provider_type"])
	}
	if parsed["name"] != "test-repo" {
		t.Errorf("unexpected name: %v", parsed["name"])
	}

	webhookConfig := parsed["webhook_config"].(map[string]interface{})
	if webhookConfig["id"] != "wh_123" {
		t.Errorf("unexpected webhook_config.id: %v", webhookConfig["id"])
	}
}

// ===========================================
// Provider Constants Tests
// ===========================================

func TestProviderConstants(t *testing.T) {
	if ProviderTypeGitHub != "github" {
		t.Errorf("unexpected ProviderTypeGitHub: %s", ProviderTypeGitHub)
	}
	if ProviderTypeGitLab != "gitlab" {
		t.Errorf("unexpected ProviderTypeGitLab: %s", ProviderTypeGitLab)
	}
	if ProviderTypeGitee != "gitee" {
		t.Errorf("unexpected ProviderTypeGitee: %s", ProviderTypeGitee)
	}
	if ProviderTypeSSH != "ssh" {
		t.Errorf("unexpected ProviderTypeSSH: %s", ProviderTypeSSH)
	}
}

// ===========================================
// Edge Cases
// ===========================================

func TestWebhookConfig_EmptyEvents(t *testing.T) {
	wc := &WebhookConfig{
		ID:     "wh_123",
		URL:    "https://example.com/webhooks",
		Events: []string{},
	}

	status := wc.ToStatus()

	if !status.Registered {
		t.Error("expected Registered to be true when ID is set")
	}
	if len(status.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(status.Events))
	}
}

func TestWebhookConfig_NilEvents(t *testing.T) {
	wc := &WebhookConfig{
		ID:     "wh_123",
		URL:    "https://example.com/webhooks",
		Events: nil,
	}

	status := wc.ToStatus()

	if !status.Registered {
		t.Error("expected Registered to be true when ID is set")
	}
	if status.Events != nil {
		t.Error("expected nil events")
	}
}

func TestRepository_NilWebhookConfig(t *testing.T) {
	repo := &Repository{
		ID:              1,
		OrganizationID:  100,
		ProviderType:    ProviderTypeGitHub,
		ProviderBaseURL: "https://github.com",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		WebhookConfig:   nil,
	}

	if repo.WebhookConfig != nil {
		t.Error("expected WebhookConfig to be nil")
	}

	var status *WebhookStatus
	if repo.WebhookConfig != nil {
		status = repo.WebhookConfig.ToStatus()
	} else {
		status = (&WebhookConfig{}).ToStatus()
	}

	if status.Registered {
		t.Error("expected empty status to have Registered=false")
	}
}

func TestRepository_Visibility(t *testing.T) {
	tests := []struct {
		visibility string
		expected   string
	}{
		{"organization", "organization"},
		{"private", "private"},
		{"public", "public"},
	}

	for _, tt := range tests {
		repo := &Repository{Visibility: tt.visibility}
		if repo.Visibility != tt.expected {
			t.Errorf("expected visibility %s, got %s", tt.expected, repo.Visibility)
		}
	}
}

func TestRepository_PreparationFields(t *testing.T) {
	script := "npm install"
	timeout := 600

	repo := &Repository{
		ID:                 1,
		OrganizationID:     100,
		ProviderType:       ProviderTypeGitHub,
		ProviderBaseURL:    "https://github.com",
		Name:               "test-repo",
		Slug:           "org/test-repo",
		PreparationScript:  &script,
		PreparationTimeout: &timeout,
	}

	if repo.PreparationScript == nil || *repo.PreparationScript != "npm install" {
		t.Error("unexpected PreparationScript")
	}
	if repo.PreparationTimeout == nil || *repo.PreparationTimeout != 600 {
		t.Error("unexpected PreparationTimeout")
	}
}

func TestRepository_SoftDelete(t *testing.T) {
	now := time.Now()
	repo := &Repository{
		ID:              1,
		OrganizationID:  100,
		ProviderType:    ProviderTypeGitHub,
		ProviderBaseURL: "https://github.com",
		Name:            "test-repo",
		Slug:        "org/test-repo",
		DeletedAt:       &now,
	}

	if repo.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
}
