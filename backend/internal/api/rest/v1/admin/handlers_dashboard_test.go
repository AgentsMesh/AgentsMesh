package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/stretchr/testify/assert"
)

func TestDashboardHandler_GetStats(t *testing.T) {
	t.Run("should get stats successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 10

		svc := adminservice.NewService(db)
		handler := NewDashboardHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/dashboard/stats", nil)

		handler.GetStats(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
