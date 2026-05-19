package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	slug := c.Param("slug")

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	userID := middleware.GetUserID(c)
	isMember, _ := h.orgService.IsMember(c.Request.Context(), org.ID, userID)
	if !isMember {
		apierr.ForbiddenAccess(c)
		return
	}

	members, err := h.orgService.ListMembers(c.Request.Context(), org.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to list members")
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	slug := c.Param("slug")

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	targetUserID := req.UserID
	if req.Email != "" {
		u, err := h.userService.GetByEmail(c.Request.Context(), req.Email)
		if err != nil {
			apierr.ResourceNotFound(c, "User not found with this email")
			return
		}
		targetUserID = u.ID
	}

	if targetUserID == 0 {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "Either email or user_id is required")
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	isMember, _ := h.orgService.IsMember(c.Request.Context(), org.ID, targetUserID)
	if isMember {
		apierr.Conflict(c, apierr.ALREADY_EXISTS, "User is already a member of this organization")
		return
	}

	if err := h.orgService.AddMember(c.Request.Context(), org.ID, targetUserID, req.Role); err != nil {
		apierr.InternalError(c, "Failed to add member")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Member added"})
}

func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	slug := c.Param("slug")
	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	if err := h.orgService.RemoveMember(c.Request.Context(), org.ID, targetUserID); err != nil {
		if errors.Is(err, organization.ErrCannotRemoveOwner) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot remove organization owner")
			return
		}
		apierr.InternalError(c, "Failed to remove member")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	slug := c.Param("slug")
	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	if err := h.orgService.UpdateMemberRole(c.Request.Context(), org.ID, targetUserID, req.Role); err != nil {
		apierr.InternalError(c, "Failed to update role")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}
