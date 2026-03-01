package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Organization Error Path Tests
// =============================================================================

func TestOrganizationHandler_GetOrganizationMembers_NotFound(t *testing.T) {
	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/999/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/invalid/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization_NotFound(t *testing.T) {
	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_ListOrganizations_WithFilters(t *testing.T) {
	t.Run("should filter by search term", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations?search=test&page=1&page_size=10", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Test member response with User field populated
func TestOrganizationHandler_GetOrganizationMembers_WithUser(t *testing.T) {
	t.Run("should return members with user info", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		testName := "Test User"
		db.members = []organization.Member{
			{
				ID:             1,
				UserID:         1,
				OrganizationID: 1,
				Role:           "owner",
				User:           &user.User{ID: 1, Email: "user@example.com", Name: &testName},
			},
		}

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Runner Error Path Tests
// =============================================================================

func TestRunnerHandler_DisableRunner_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/invalid/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_EnableRunner_NotFound(t *testing.T) {
	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/999/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/invalid/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner_NotFound(t *testing.T) {
	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_GetRunner_InvalidID(t *testing.T) {
	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRunnerHandler_GetRunner_OrgNotFound(t *testing.T) {
	t.Run("should return 200 when runner found but org not found", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 999}
		// Organization not found, but runner exists - should still return runner info

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
