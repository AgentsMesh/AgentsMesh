package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
	"github.com/gin-gonic/gin"
)

func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	orgs, err := h.orgService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to list organizations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := slugkit.Validate(req.Slug); err != nil {
		suggestion, _ := slugkit.SanitizeAndValidate(req.Slug)
		apierr.RespondWithExtra(c, http.StatusBadRequest, apierr.VALIDATION_FAILED,
			"Slug must contain only lowercase letters, numbers, and hyphens, and must start and end with alphanumeric characters",
			gin.H{"field": "slug", "suggestion": suggestion})
		return
	}

	userID := middleware.GetUserID(c)

	org, err := h.orgService.Create(c.Request.Context(), userID, &organization.CreateRequest{
		Name:    req.Name,
		Slug:    req.Slug,
		LogoURL: req.LogoURL,
	})

	if err != nil {
		if errors.Is(err, organization.ErrSlugAlreadyExists) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Organization slug already exists")
			return
		}
		apierr.InternalError(c, "Failed to create organization")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": org})
}

func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	slug := c.Param("slug")
	if slugkit.IsReserved(slug) {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	slug := c.Param("slug")
	if slugkit.IsReserved(slug) {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	var req UpdateOrganizationRequest
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

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.LogoURL != "" {
		updates["logo_url"] = req.LogoURL
	}

	org, err = h.orgService.Update(c.Request.Context(), org.ID, updates)
	if err != nil {
		apierr.InternalError(c, "Failed to update organization")
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	slug := c.Param("slug")
	if slugkit.IsReserved(slug) {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	org, err := h.orgService.GetOrgBySlug(c.Request.Context(), slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Organization not found")
		return
	}

	userID := middleware.GetUserID(c)
	isOwner, _ := h.orgService.IsOwner(c.Request.Context(), org.ID, userID)
	if !isOwner {
		apierr.ForbiddenOwner(c)
		return
	}

	if err := h.orgService.Delete(c.Request.Context(), org.ID); err != nil {
		apierr.InternalError(c, "Failed to delete organization")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted"})
}
