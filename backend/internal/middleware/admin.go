package middleware

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func AdminMiddleware(db database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
			return
		}

		var u user.User
		if err := db.First(&u, userID); err != nil {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "User not found")
			return
		}

		if !u.IsSystemAdmin {
			apierr.AbortForbidden(c, apierr.SYSTEM_ADMIN_REQUIRED, "Access denied: system administrator privileges required")
			return
		}

		if !u.IsActive {
			apierr.AbortForbidden(c, apierr.ACCOUNT_DISABLED, "Access denied: user account is disabled")
			return
		}

		c.Set("admin_user", &u)
		c.Set("admin_user_id", u.ID)

		c.Next()
	}
}

func GetAdminUser(c *gin.Context) *user.User {
	if u, exists := c.Get("admin_user"); exists {
		if adminUser, ok := u.(*user.User); ok {
			return adminUser
		}
	}
	return nil
}

func GetAdminUserID(c *gin.Context) int64 {
	if id, exists := c.Get("admin_user_id"); exists {
		if adminID, ok := id.(int64); ok {
			return adminID
		}
	}
	return 0
}
