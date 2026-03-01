package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ===========================================
// RegisterRepositoryWebhook Tests
// ===========================================

func TestRegisterRepositoryWebhook_InvalidID(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/invalid/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Invalid repository ID" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	if _, ok := resp["code"]; !ok {
		t.Error("expected 'code' field in error response")
	}
}

func TestRegisterRepositoryWebhook_Forbidden_MemberRole(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Admin permission required" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	if _, ok := resp["code"]; !ok {
		t.Error("expected 'code' field in error response")
	}
}

func TestRegisterRepositoryWebhook_RepoNotFound(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/999/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestRegisterRepositoryWebhook_OrgMismatch(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	// Repo belongs to org 2
	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 2, // Different org
		Name:           "test-repo",
	})

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner") // User is in org 1
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Access denied" {
		t.Errorf("unexpected error: %v", resp["error"])
	}
	if _, ok := resp["code"]; !ok {
		t.Error("expected 'code' field in error response")
	}
}

func TestRegisterRepositoryWebhook_NoWebhookService(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.webhookService = nil // No webhook service
	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestRegisterRepositoryWebhook_Success(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
		ProviderType:   "gitlab",
	})

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	result := resp["result"].(map[string]interface{})
	if !result["registered"].(bool) {
		t.Error("expected registered to be true")
	}
	if result["webhook_id"] != "wh_test123" {
		t.Errorf("unexpected webhook_id: %v", result["webhook_id"])
	}
}

func TestRegisterRepositoryWebhook_NeedsManualSetup(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.registerResult = &repository.WebhookResult{
		RepoID:              1,
		Registered:          false,
		NeedsManualSetup:    true,
		ManualWebhookURL:    "https://example.com/webhooks/test-org/gitlab/1",
		ManualWebhookSecret: "secret123",
		Error:               "No OAuth token available",
	}

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	result := resp["result"].(map[string]interface{})
	if result["needs_manual_setup"] != true {
		t.Error("expected needs_manual_setup to be true")
	}
	if result["manual_webhook_url"] != "https://example.com/webhooks/test-org/gitlab/1" {
		t.Errorf("unexpected manual_webhook_url: %v", result["manual_webhook_url"])
	}
}

func TestRegisterRepositoryWebhook_RegisterError(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.registerError = errors.New("registration failed")

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
