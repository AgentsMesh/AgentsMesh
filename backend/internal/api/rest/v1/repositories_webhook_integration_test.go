package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Integration-like Tests
// ===========================================

func TestWebhookWorkflow_RegisterThenStatus(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
		ProviderType:   "gitlab",
	})

	// First register the webhook
	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("register failed: %d", w.Code)
	}

	// Now update mock to return registered status
	mockSvc.webhookService.webhookStatus = &gitprovider.WebhookStatus{
		Registered: true,
		WebhookID:  "wh_test123",
		IsActive:   true,
	}

	// Then check status
	router2 := gin.New()
	router2.GET("/api/v1/orgs/:slug/repositories/:id/webhook/status", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.GetRepositoryWebhookStatus(c)
	})

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/status", nil)
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status check failed: %d", w2.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	status := resp["webhook_status"].(map[string]interface{})
	if !status["registered"].(bool) {
		t.Error("expected registered to be true after registration")
	}
}

func TestWebhookWorkflow_ManualSetup(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
		ProviderType:   "gitlab",
		WebhookConfig: &gitprovider.WebhookConfig{
			URL:              "https://example.com/webhooks/test-org/gitlab/1",
			Secret:           "secret123",
			Events:           []string{"merge_request", "pipeline"},
			NeedsManualSetup: true,
		},
	})

	// Register returns needs manual setup
	mockSvc.webhookService.registerResult = &repository.WebhookResult{
		RepoID:              1,
		Registered:          false,
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example.com/webhooks/test-org/gitlab/1",
		ManualWebhookSecret: "secret123",
	}
	mockSvc.webhookService.webhookSecret = "secret123"

	// 1. Register webhook (gets manual setup result)
	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	result := resp["result"].(map[string]interface{})
	if !result["needs_manual_setup"].(bool) {
		t.Error("expected needs_manual_setup")
	}

	// 2. Get secret for manual configuration
	router2 := gin.New()
	router2.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.GetRepositoryWebhookSecret(c)
	})

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/secret", nil)
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("get secret failed: %d", w2.Code)
	}

	var resp2 map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2["webhook_secret"] != "secret123" {
		t.Errorf("unexpected secret: %v", resp2["webhook_secret"])
	}

	// 3. Mark as configured
	router3 := gin.New()
	router3.POST("/api/v1/orgs/:slug/repositories/:id/webhook/configured", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.MarkRepositoryWebhookConfigured(c)
	})

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook/configured", nil)
	w3 := httptest.NewRecorder()
	router3.ServeHTTP(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("mark configured failed: %d", w3.Code)
	}
}
