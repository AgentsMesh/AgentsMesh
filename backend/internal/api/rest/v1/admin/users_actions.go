package admin

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// DisableUser disables a user account
func (h *UserHandler) DisableUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	// Prevent disabling self
	adminUserID := middleware.GetAdminUserID(c)
	if userID == adminUserID {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot disable your own account")
		return
	}

	// Get old data for audit log
	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.DisableUser(c.Request.Context(), userID)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to disable user")
		return
	}

	// Log disable action
	h.logAction(c, admin.AuditActionUserDisable, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

// EnableUser enables a user account
func (h *UserHandler) EnableUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	// Get old data for audit log
	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.EnableUser(c.Request.Context(), userID)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to enable user")
		return
	}

	// Log enable action
	h.logAction(c, admin.AuditActionUserEnable, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

// GrantAdmin grants system admin privileges to a user
func (h *UserHandler) GrantAdmin(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	// Get old data for audit log
	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.GrantAdmin(c.Request.Context(), userID)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to grant admin privileges")
		return
	}

	// Log grant admin action
	h.logAction(c, admin.AuditActionUserGrantAdmin, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

// RevokeAdmin revokes system admin privileges from a user
func (h *UserHandler) RevokeAdmin(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	adminUserID := middleware.GetAdminUserID(c)

	// Get old data for audit log
	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.RevokeAdmin(c.Request.Context(), userID, adminUserID)
	if err != nil {
		if err == adminservice.ErrUserNotFound {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		if err == adminservice.ErrCannotRevokeOwnAdmin {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot revoke your own admin privileges")
			return
		}
		apierr.InternalError(c, "Failed to revoke admin privileges")
		return
	}

	// Log revoke admin action
	h.logAction(c, admin.AuditActionUserRevokeAdmin, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}
