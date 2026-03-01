package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/gitprovider"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Test Setup
// ===========================================

func init() {
	gin.SetMode(gin.TestMode)
}

func setupWebhookHandlerTest() (*RepositoryHandler, *mockRepositoryService, *gin.Engine) {
	mockRepoSvc := newMockRepositoryService()
	handler := NewRepositoryHandler(mockRepoSvc)
	router := gin.New()
	return handler, mockRepoSvc, router
}

func setTenantContext(c *gin.Context, orgID int64, orgSlug string, userRole string) {
	tenant := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: orgSlug,
		UserID:           1,
		UserRole:         userRole,
	}
	c.Set("tenant", tenant)
	c.Set("user_id", int64(1))
}

// ===========================================
// Permission Tests - Admin/Owner Allowed
// ===========================================

func TestWebhookEndpoints_AdminRoleAllowed(t *testing.T) {
	handler, mockSvc, router := setupWebhookHandlerTest()

	mockSvc.AddRepo(&gitprovider.Repository{
		ID:             1,
		OrganizationID: 1,
		Name:           "test-repo",
	})

	router.POST("/api/v1/orgs/:slug/repositories/:id/webhook", func(c *gin.Context) {
		setTenantContext(c, 1, "test-org", "admin")
		handler.RegisterRepositoryWebhook(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/test-org/repositories/1/webhook", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Admin role should be allowed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for admin role, got %d", w.Code)
	}
}

func TestWebhookEndpoints_OwnerRoleAllowed(t *testing.T) {
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

	// Owner role should be allowed
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for owner role, got %d", w.Code)
	}
}
