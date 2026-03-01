package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestOrganizationHandler_ListOrganizations(t *testing.T) {
	t.Run("should list organizations successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1", Slug: "org-1"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations", nil)

		handler.ListOrganizations(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOrganizationHandler_DeleteOrganization(t *testing.T) {
	t.Run("should delete organization successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 0

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 409 when organization has runners", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}
		db.runnerCount = 5

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteOrganization(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestOrganizationHandler_GetOrganization(t *testing.T) {
	t.Run("should get organization successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when organization not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("should return 400 for invalid ID", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewOrganizationHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/organizations/invalid", nil)
		c.Params = gin.Params{{Key: "id", Value: "invalid"}}

		handler.GetOrganization(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestOrganizationHandler_GetOrganizationMembers(t *testing.T) {
	t.Run("should get organization members successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org", Slug: "test-org"}
		db.members = []organization.Member{
			{ID: 1, UserID: 1, OrganizationID: 1, Role: "owner"},
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

