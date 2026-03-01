package middleware

import (
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/database"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware validates that the authenticated user is a system admin.
// This middleware must be used after AuthMiddleware.
func AdminMiddleware(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by AuthMiddleware)
		userID, exists := c.Get("user_id")
		if !exists {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
			return
		}

		// Fetch user from database to verify is_system_admin flag
		var u user.User
		if err := db.First(&u, userID); err != nil {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "User not found")
			return
		}

		// Verify user is a system admin
		if !u.IsSystemAdmin {
			apierr.AbortForbidden(c, apierr.SYSTEM_ADMIN_REQUIRED, "Access denied: system administrator privileges required")
			return
		}

		// Verify user is active
		if !u.IsActive {
			apierr.AbortForbidden(c, apierr.ACCOUNT_DISABLED, "Access denied: user account is disabled")
			return
		}

		// Set admin user in context for audit logging
		c.Set("admin_user", &u)
		c.Set("admin_user_id", u.ID)

		c.Next()
	}
}

// GetAdminUser retrieves the admin user from the context
func GetAdminUser(c *gin.Context) *user.User {
	if u, exists := c.Get("admin_user"); exists {
		if adminUser, ok := u.(*user.User); ok {
			return adminUser
		}
	}
	return nil
}

// GetAdminUserID retrieves the admin user ID from the context
func GetAdminUserID(c *gin.Context) int64 {
	if id, exists := c.Get("admin_user_id"); exists {
		if adminID, ok := id.(int64); ok {
			return adminID
		}
	}
	return 0
}
