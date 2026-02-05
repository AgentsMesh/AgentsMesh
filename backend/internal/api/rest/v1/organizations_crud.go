package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/gin-gonic/gin"
)

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
