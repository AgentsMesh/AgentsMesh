package admin

import (
	"log"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"
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
		log.Printf("[AUDIT] Warning: admin user ID not found in context for action %s", action)
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
		log.Printf("[AUDIT] Failed to log action %s on %s/%d by admin %d: %v",
			action, targetType, targetID, adminUserID, err)
	}
}
