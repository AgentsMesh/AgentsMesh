package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/gin-gonic/gin"
)

// ===========================================
// GetRepositoryWebhookStatus Tests
// ===========================================

func TestGetRepositoryWebhookStatus_InvalidID(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/status", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.GetRepositoryWebhookStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/invalid/webhook/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetRepositoryWebhookStatus_Success(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.webhookStatus = &gitprovider.WebhookStatus{
		Registered: true,
		WebhookID:  "wh_123",
		WebhookURL: "https://example.com/webhooks/test-org/gitlab/1",
		Events:     []string{"merge_request", "pipeline"},
		IsActive:   true,
	}

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/status", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.GetRepositoryWebhookStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	status := resp["webhook_status"].(map[string]interface{})
	if status["registered"] != true {
		t.Error("expected registered to be true")
	}
	if status["webhook_id"] != "wh_123" {
		t.Errorf("unexpected webhook_id: %v", status["webhook_id"])
	}
}

func TestGetRepositoryWebhookStatus_NotRegistered(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	// Default mock returns Registered: false

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/status", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.GetRepositoryWebhookStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	status := resp["webhook_status"].(map[string]interface{})
	if status["registered"] != false {
		t.Error("expected registered to be false")
	}
}
