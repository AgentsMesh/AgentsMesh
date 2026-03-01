package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRunnerHandler_ListRunners(t *testing.T) {
	t.Run("should list runners successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "node-1", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Org 1"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRunnerHandler_DisableRunner(t *testing.T) {
	t.Run("should disable runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: true}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/999/disable", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.DisableRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRunnerHandler_DeleteRunner(t *testing.T) {
	t.Run("should delete runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 0

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 409 when runner has active pods", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node"}
		db.activePodCount = 3

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("DELETE", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.DeleteRunner(c)

		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestRunnerHandler_GetRunner(t *testing.T) {
	t.Run("should get runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", OrganizationID: 1}
		db.organizations[1] = &organization.Organization{ID: 1, Name: "Test Org"}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 404 when runner not found", func(t *testing.T) {
		db := newMockHandlerDB()

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners/999", nil)
		c.Params = gin.Params{{Key: "id", Value: "999"}}

		handler.GetRunner(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRunnerHandler_EnableRunner(t *testing.T) {
	t.Run("should enable runner successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", IsEnabled: false}

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("POST", "/runners/1/enable", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.EnableRunner(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRunnerHandler_ListRunners_WithFilters(t *testing.T) {
	t.Run("should filter by search and status", func(t *testing.T) {
		db := newMockHandlerDB()
		db.runners[1] = &runner.Runner{ID: 1, NodeID: "test-node", Status: "online"}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewRunnerHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/runners?search=test&status=online&org_id=1&page=1&page_size=10", nil)

		handler.ListRunners(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
