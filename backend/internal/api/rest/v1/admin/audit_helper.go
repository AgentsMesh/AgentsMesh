package admin

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
)

// LogAdminAction logs an audit action with proper error handling.
// This is a shared helper function used by all admin handlers.
func LogAdminAction(
	c *gin.Context,
	svc *adminservice.Service,
	action admin.AuditAction,
	targetType admin.TargetType,
	targetID int64,
	oldData interface{},
	newData interface{},
) {
	adminUserID := middleware.GetAdminUserID(c)
	if adminUserID == 0 {
		slog.Warn("admin user ID not found in context for audit action",
			"action", action)
		return
	}

	err := svc.LogActionFromContext(
		c.Request.Context(),
		adminUserID,
		action,
		targetType,
		targetID,
		oldData,
		newData,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
	)

	if err != nil {
		// Log the error but don't fail the request
		// Audit logging failure should not prevent the operation from succeeding
		slog.Warn("failed to log admin audit action",
			"action", action, "target_type", targetType, "target_id", targetID,
			"admin_id", adminUserID, "error", err)
	}
}
