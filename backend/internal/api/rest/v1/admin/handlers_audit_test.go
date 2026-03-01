package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAuditLogHandler_ListAuditLogs(t *testing.T) {
	t.Run("should list audit logs successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		db.auditLogs = []admin.AuditLog{
			{ID: 1, AdminUserID: 1, Action: admin.AuditActionUserView},
		}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by admin_user_id", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?admin_user_id=1", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by action", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?action=user.view", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by target_type and target_id", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?target_type=user&target_id=1", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should filter by date range", func(t *testing.T) {
		db := newMockHandlerDB()
		db.totalCount = 0

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.countErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("should include audit log with old_data and new_data", func(t *testing.T) {
		db := newMockHandlerDB()
		oldData := `{"is_active": true}`
		newData := `{"is_active": false}`
		testUser := &user.User{ID: 1, Email: "admin@example.com"}
		db.auditLogs = []admin.AuditLog{
			{
				ID:          1,
				AdminUserID: 1,
				Action:      admin.AuditActionUserDisable,
				OldData:     &oldData,
				NewData:     &newData,
				AdminUser:   testUser,
			},
		}
		db.totalCount = 1

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestLogAdminAction(t *testing.T) {
	t.Run("should log action successfully", func(t *testing.T) {
		db := newMockHandlerDB()
		svc := adminservice.NewService(db)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("User-Agent", "Test Agent")

		LogAdminAction(c, svc, admin.AuditActionUserView, admin.TargetTypeUser, 1, nil, nil)

		assert.Len(t, db.auditLogs, 1)
		assert.Equal(t, admin.AuditActionUserView, db.auditLogs[0].Action)
	})

	t.Run("should handle missing admin user ID", func(t *testing.T) {
		db := newMockHandlerDB()
		svc := adminservice.NewService(db)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		// No admin_user_id set

		// Should not panic
		LogAdminAction(c, svc, admin.AuditActionUserView, admin.TargetTypeUser, 1, nil, nil)

		// No log should be created
		assert.Len(t, db.auditLogs, 0)
	})
}

func TestAuditLogHandler_ListAuditLogs_InternalError(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		db := newMockHandlerDB()
		db.findErr = gorm.ErrInvalidDB

		svc := adminservice.NewService(db)
		handler := NewAuditLogHandler(svc)

		w := httptest.NewRecorder()
		c := createAdminContext(w)
		c.Request = httptest.NewRequest("GET", "/audit-logs", nil)

		handler.ListAuditLogs(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
