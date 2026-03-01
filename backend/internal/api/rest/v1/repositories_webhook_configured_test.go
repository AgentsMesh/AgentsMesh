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
// MarkRepositoryWebhookConfigured Tests
// ===========================================

func TestMarkRepositoryWebhookConfigured_InvalidID(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook/configured", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.MarkRepositoryWebhookConfigured(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/invalid/webhook/configured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMarkRepositoryWebhookConfigured_Forbidden_MemberRole(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook/configured", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.MarkRepositoryWebhookConfigured(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook/configured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestMarkRepositoryWebhookConfigured_Success(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook/configured", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.MarkRepositoryWebhookConfigured(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook/configured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Webhook marked as configured" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestMarkRepositoryWebhookConfigured_NotFound(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.markConfiguredErr = repository.ErrWebhookNotFound

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook/configured", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.MarkRepositoryWebhookConfigured(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook/configured", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
