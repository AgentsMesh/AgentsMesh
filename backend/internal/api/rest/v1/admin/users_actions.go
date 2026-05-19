package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *UserHandler) DisableUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	adminUserID := middleware.GetAdminUserID(c)
	if userID == adminUserID {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot disable your own account")
		return
	}

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.DisableUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to disable user")
		return
	}

	h.logAction(c, admin.AuditActionUserDisable, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *UserHandler) EnableUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.EnableUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to enable user")
		return
	}

	h.logAction(c, admin.AuditActionUserEnable, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *UserHandler) GrantAdmin(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.GrantAdmin(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to grant admin privileges")
		return
	}

	h.logAction(c, admin.AuditActionUserGrantAdmin, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *UserHandler) RevokeAdmin(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	adminUserID := middleware.GetAdminUserID(c)

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.RevokeAdmin(c.Request.Context(), userID, adminUserID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		if errors.Is(err, adminservice.ErrCannotRevokeOwnAdmin) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot revoke your own admin privileges")
			return
		}
		apierr.InternalError(c, "Failed to revoke admin privileges")
		return
	}

	h.logAction(c, admin.AuditActionUserRevokeAdmin, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *UserHandler) VerifyUserEmail(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.VerifyUserEmail(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to verify user email")
		return
	}

	h.logAction(c, admin.AuditActionUserVerifyEmail, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}

func (h *UserHandler) UnverifyUserEmail(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	oldUser, _ := h.adminService.GetUser(c.Request.Context(), userID)

	user, err := h.adminService.UnverifyUserEmail(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, adminservice.ErrUserNotFound) {
			apierr.ResourceNotFound(c, "User not found")
			return
		}
		apierr.InternalError(c, "Failed to unverify user email")
		return
	}

	h.logAction(c, admin.AuditActionUserUnverifyEmail, admin.TargetTypeUser, userID, oldUser, user)

	c.JSON(http.StatusOK, adminUserResponse(user))
}
