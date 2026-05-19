package admin

import (
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/gin-gonic/gin"
)

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
		slog.WarnContext(c.Request.Context(), "admin user ID not found in context for audit action",
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
		slog.WarnContext(c.Request.Context(), "failed to log admin audit action",
			"action", action, "target_type", targetType, "target_id", targetID,
			"admin_id", adminUserID, "error", err)
	}
}
