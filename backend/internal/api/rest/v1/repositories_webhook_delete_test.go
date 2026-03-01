package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/gin-gonic/gin"
)

// ===========================================
// DeleteRepositoryWebhook Tests
// ===========================================

func TestDeleteRepositoryWebhook_InvalidID(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.DELETE("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.DeleteRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/test-org/repositories/invalid/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteRepositoryWebhook_Forbidden_MemberRole(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.DELETE("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.DeleteRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestDeleteRepositoryWebhook_Success(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})

	router.DELETE("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.DeleteRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Webhook deleted" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestDeleteRepositoryWebhook_DeleteError(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.deleteError = errors.New("delete failed")

	router.DELETE("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.DeleteRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
