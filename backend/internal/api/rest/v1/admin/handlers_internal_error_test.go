package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// =============================================================================
// Dashboard Internal Error Tests
// =============================================================================

func TestDashboardHandler_GetStats_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewDashboardHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/dashboard/stats", nil)

		handler.GetStats(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// User Internal Error Tests
// =============================================================================

func TestUserHandler_ListUsers_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users", nil)

		handler.ListUsers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_GetUser_InternalError(t *testing.T) {
	t.Run("should return 404 when service fails with internal error", func(t *testing.T) {
		db := newMockHandlerDB()
		db.firstErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetUser(c)

		// When firstErr is set, it returns ErrUserNotFound which maps to 404
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUserHandler_UpdateUser_InternalError(t *testing.T) {
	t.Run("should return 500 when update fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[1] = &user.User{ID: 1, Email: "test@example.com"}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		body := bytes.NewBufferString(`{"name": "New Name"}`)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("PUT", "/users/1", body)
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_DisableUser_InternalError(t *testing.T) {
	t.Run("should return 500 when disable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: true}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.DisableUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_EnableUser_InternalError(t *testing.T) {
	t.Run("should return 500 when enable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsActive: false}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.EnableUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_GrantAdmin_InternalError(t *testing.T) {
	t.Run("should return 500 when grant admin fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "user@example.com", IsSystemAdmin: false}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/grant-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.GrantAdmin(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestUserHandler_RevokeAdmin_InternalError(t *testing.T) {
	t.Run("should return 500 when revoke admin fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.users[2] = &user.User{ID: 2, Email: "other@example.com", IsSystemAdmin: true}
		db.updatesErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewUserHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/users/2/revoke-admin", nil)
		c.Params = gin.Params{{Key: "id", Value: "2"}}

		handler.RevokeAdmin(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// Organization Internal Error Tests
// =============================================================================

func TestOrganizationHandler_ListOrganizations_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrganizationHandler_GetOrganization_InternalError(t *testing.T) {
	t.Run("should handle internal error scenario", func(t *testing.T) {
		db := newMockHandlerDB()
		// Setting firstErr causes GetOrganization to return ErrOrganizationNotFound which maps to 404
		db.firstErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization_InternalError(t *testing.T) {
	t.Run("should return 500 when delete fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0
		db.deleteErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestOrganizationHandler_GetOrganizationMembers_InternalError(t *testing.T) {
	t.Run("should return 500 when get members fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.findErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1/members", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganizationMembers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// =============================================================================
// Runner Internal Error Tests
// =============================================================================

func TestRunnerHandler_ListRunners_Error(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_DisableRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when disable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: true}
		db.saveErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_EnableRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when enable fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: false}
		db.saveErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner_InternalError(t *testing.T) {
	t.Run("should return 500 when delete fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0
		db.deleteErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
