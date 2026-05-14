package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
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
