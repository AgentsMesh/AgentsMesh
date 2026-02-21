package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/gin-gonic/gin"
)

// ListMembers lists organization members
// GET /api/v1/organizations/:slug/members
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	slug := c.Param("slug")

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check membership
	userID := middleware.GetUserID(c)
	isMember, _ := h.orgService.IsMember(c.Request.Context(), org.ID, userID)
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	members, err := h.orgService.ListMembers(c.Request.Context(), org.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// InviteMember invites a member to organization
// POST /api/v1/organizations/:slug/members
// Supports both email-based invitation and direct user_id addition
func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	slug := c.Param("slug")

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve target user ID: either from email or direct user_id
	targetUserID := req.UserID
	if req.Email != "" {
		u, err := h.userService.GetByEmail(c.Request.Context(), req.Email)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found with this email"})
			return
		}
		targetUserID = u.ID
	}

	if targetUserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either email or user_id is required"})
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check admin permission
	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	// Check if already a member
	isMember, _ := h.orgService.IsMember(c.Request.Context(), org.ID, targetUserID)
	if isMember {
		c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this organization"})
		return
	}

	if err := h.orgService.AddMember(c.Request.Context(), org.ID, targetUserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Member added"})
}

// RemoveMember removes a member from organization
// DELETE /api/v1/organizations/:slug/members/:user_id
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	slug := c.Param("slug")
	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check admin permission
	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.orgService.RemoveMember(c.Request.Context(), org.ID, targetUserID); err != nil {
		if err == organization.ErrCannotRemoveOwner {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove organization owner"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// UpdateMemberRole updates a member's role
// PUT /api/v1/organizations/:slug/members/:user_id
func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	slug := c.Param("slug")
	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check admin permission
	userID := middleware.GetUserID(c)
	isAdmin, _ := h.orgService.IsAdmin(c.Request.Context(), org.ID, userID)
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	if err := h.orgService.UpdateMemberRole(c.Request.Context(), org.ID, targetUserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}
