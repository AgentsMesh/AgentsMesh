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
// GetRepositoryWebhookSecret Tests
// ===========================================

func TestGetRepositoryWebhookSecret_InvalidID(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.GetRepositoryWebhookSecret(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/invalid/webhook/secret", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetRepositoryWebhookSecret_Forbidden_MemberRole(t *testing.T) {
	handler, _, router := setupWebhookHandlerTest()

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "member")
		handler.GetRepositoryWebhookSecret(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/secret", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestGetRepositoryWebhookSecret_Success(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
		WebhookConfig: &gitprovider.WebhookConfig{
			URL:              "https://example.com/webhooks/test-org/gitlab/1",
			Secret:           "secret123",
			Events:           []string{"merge_request", "pipeline"},
			NeedsManualSetup: true,
		},
	})
	mockSvc.webhookService.webhookSecret = "secret123"

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "admin")
		handler.GetRepositoryWebhookSecret(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/secret", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["webhook_secret"] != "secret123" {
		t.Errorf("unexpected webhook_secret: %v", resp["webhook_secret"])
	}
}

func TestGetRepositoryWebhookSecret_NotFound(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.secretError = repository.ErrWebhookNotFound

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.GetRepositoryWebhookSecret(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/secret", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGetRepositoryWebhookSecret_AlreadyRegistered(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})
	mockSvc.webhookService.secretError = errors.New("webhook is already automatically registered, no manual setup required")

	router.GET("/api/v1/orgs/:slug/repositories/:id/webhook/secret", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "owner")
		handler.GetRepositoryWebhookSecret(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/test-org/repositories/1/webhook/secret", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
