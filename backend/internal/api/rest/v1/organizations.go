package v1

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/gin-gonic/gin"
)

// slugRegex validates organization slug: lowercase letters, numbers, and hyphens
// Must start and end with alphanumeric, no consecutive hyphens
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// OrganizationHandler handles organization-related requests
type OrganizationHandler struct {
	orgService *organization.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *organization.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganizationRequest represents organization creation request
type CreateOrganizationRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Slug    string `json:"slug" binding:"required,min=2,max=100"`
	LogoURL string `json:"logo_url"`
}

// ListOrganizations lists user's organizations
// GET /api/v1/organizations
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orgs, err := h.orgService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list organizations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// CreateOrganization creates a new organization
// POST /api/v1/organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate slug format: lowercase alphanumeric with hyphens
	if !slugRegex.MatchString(req.Slug) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slug must contain only lowercase letters, numbers, and hyphens, and must start and end with alphanumeric characters"})
		return
	}

	userID := middleware.GetUserID(c)

	org, err := h.orgService.Create(c.Request.Context(), userID, &organization.CreateRequest{
		Name:    req.Name,
		Slug:    req.Slug,
		LogoURL: req.LogoURL,
	})

	if err != nil {
		if err == organization.ErrSlugAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Organization slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": org})
}

// GetOrganization returns organization by slug
// GET /api/v1/organizations/:slug
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// UpdateOrganizationRequest represents organization update request
type UpdateOrganizationRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

// UpdateOrganization updates an organization
// PUT /api/v1/organizations/:slug
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	slug := c.Param("slug")

	var req UpdateOrganizationRequest
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

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}

	org, err = h.orgService.Update(c.Request.Context(), org.ID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// DeleteOrganization deletes an organization
// DELETE /api/v1/organizations/:slug
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	slug := c.Param("slug")

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	// Check owner permission
	userID := middleware.GetUserID(c)
	isOwner, _ := h.orgService.IsOwner(c.Request.Context(), org.ID, userID)
	if !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Owner permission required"})
		return
	}

	if err := h.orgService.Delete(c.Request.Context(), org.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted"})
}

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

// InviteMemberRequest represents member invitation request
type InviteMemberRequest struct {
	UserID int64  `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}

// InviteMember invites a member to organization
// POST /api/v1/organizations/:slug/members
func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	slug := c.Param("slug")

	var req InviteMemberRequest
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

	if err := h.orgService.AddMember(c.Request.Context(), org.ID, req.UserID, req.Role); err != nil {
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

// UpdateMemberRoleRequest represents role update request
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
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
