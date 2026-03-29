package repository

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

// ===========================================
// WebhookService Dependency Injection Tests
// These tests verify the SetWebhookService/GetWebhookService pattern
// ===========================================

func TestNewService_WebhookServiceNil(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))

	// Initially, webhook service should be nil
	ws := service.GetWebhookService()
	if ws != nil {
		t.Error("expected webhook service to be nil initially")
	}
}

func TestSetWebhookService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}

	webhookService := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)

	// Set webhook service
	service.SetWebhookService(webhookService)

	// Verify it was set
	ws := service.GetWebhookService()
	if ws == nil {
		t.Fatal("expected webhook service to be set")
	}
}

func TestSetWebhookService_CanBeCalledMultipleTimes(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}

	// First webhook service
	webhookService1 := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService1)

	ws1 := service.GetWebhookService()
	if ws1 == nil {
		t.Fatal("expected first webhook service to be set")
	}

	// Replace with second webhook service
	webhookService2 := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService2)

	ws2 := service.GetWebhookService()
	if ws2 == nil {
		t.Fatal("expected second webhook service to be set")
	}

	// Note: ws1 and ws2 will be different instances
	// We can't directly compare them since GetWebhookService returns interface
}

func TestSetWebhookService_NilValue(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}

	// Set a webhook service first
	webhookService := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService)

	ws := service.GetWebhookService()
	if ws == nil {
		t.Fatal("expected webhook service to be set")
	}

	// Set to nil (clear the webhook service)
	service.SetWebhookService(nil)

	ws = service.GetWebhookService()
	if ws != nil {
		t.Error("expected webhook service to be nil after setting to nil")
	}
}

func TestGetWebhookService_ReturnsInterface(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}

	webhookService := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService)

	// GetWebhookService returns WebhookServiceInterface
	ws := service.GetWebhookService()
	if ws == nil {
		t.Fatal("expected webhook service interface")
	}

	// Verify it implements the interface methods (compile-time check)
	var _ WebhookServiceInterface = ws
}

// ===========================================
// CreateWithWebhook Tests - Webhook Service Availability
// ===========================================

func TestCreateWithWebhook_NoWebhookService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db)) // No webhook service set

	userID := int64(1)
	req := &CreateRequest{
		OrganizationID:   1,
		ProviderType:     "gitlab",
		ProviderBaseURL:  "https://gitlab.com",
		CloneURL:         "https://gitlab.com/org/test-repo.git",
		ExternalID:       "12345",
		Name:             "test-repo",
		Slug:         "org/test-repo-nowh",
		Visibility:       "organization",
		ImportedByUserID: &userID,
	}

	repo, webhookResult, err := service.CreateWithWebhook(nil, req, "test-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repository should still be created
	if repo == nil {
		t.Fatal("expected repository to be created")
	}
	if repo.Name != "test-repo" {
		t.Errorf("expected name 'test-repo', got %s", repo.Name)
	}

	// Webhook result should be nil since no webhook service is configured
	if webhookResult != nil {
		t.Errorf("expected nil webhook result when no webhook service, got %+v", webhookResult)
	}
}

func TestCreateWithWebhook_NoUserID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}

	webhookService := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService)

	req := &CreateRequest{
		OrganizationID:   1,
		ProviderType:     "gitlab",
		ProviderBaseURL:  "https://gitlab.com",
		CloneURL:         "https://gitlab.com/org/test-repo.git",
		ExternalID:       "67890",
		Name:             "test-repo",
		Slug:         "org/test-repo-nouser",
		Visibility:       "organization",
		ImportedByUserID: nil, // No user ID
	}

	repo, webhookResult, err := service.CreateWithWebhook(nil, req, "test-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repository should still be created
	if repo == nil {
		t.Fatal("expected repository to be created")
	}

	// Webhook result should be nil since no user ID is provided
	if webhookResult != nil {
		t.Errorf("expected nil webhook result when no user ID, got %+v", webhookResult)
	}
}

func TestCreateWithWebhook_WithWebhookService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(infra.NewGitProviderRepository(db))
	cfg := &config.Config{}
	cfg.PrimaryDomain = "app.example.com"
	cfg.UseHTTPS = true

	webhookService := NewWebhookService(infra.NewGitProviderRepository(db), cfg, nil, nil)
	service.SetWebhookService(webhookService)

	userID := int64(1)
	req := &CreateRequest{
		OrganizationID:   1,
		ProviderType:     "gitlab",
		ProviderBaseURL:  "https://gitlab.com",
		CloneURL:         "https://gitlab.com/org/test-repo.git",
		ExternalID:       "11111",
		Name:             "test-repo",
		Slug:         "org/test-repo-withwh",
		Visibility:       "organization",
		ImportedByUserID: &userID,
	}

	repo, webhookResult, err := service.CreateWithWebhook(nil, req, "test-org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repository should be created
	if repo == nil {
		t.Fatal("expected repository to be created")
	}

	// Webhook result should indicate registration is in progress
	// (async registration - the actual result is not available immediately)
	if webhookResult == nil {
		t.Fatal("expected webhook result")
	}
	if webhookResult.RepoID != repo.ID {
		t.Errorf("expected webhook result RepoID %d, got %d", repo.ID, webhookResult.RepoID)
	}
	// The result should indicate in-progress state
	if webhookResult.Error != "Webhook registration in progress" {
		t.Logf("webhook result: %+v", webhookResult)
	}
}
