package admin

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// OrganizationHandler handles organization management requests
type OrganizationHandler struct {
	adminService *adminservice.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(adminSvc *adminservice.Service) *OrganizationHandler {
	return &OrganizationHandler{
		adminService: adminSvc,
	}
}

// RegisterRoutes registers organization management routes
func (h *OrganizationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	orgsGroup := rg.Group("/organizations")
	{
		orgsGroup.GET("", h.ListOrganizations)
		orgsGroup.GET("/:id", h.GetOrganization)
		orgsGroup.GET("/:id/members", h.GetOrganizationMembers)
		orgsGroup.DELETE("/:id", h.DeleteOrganization)
	}
}

// ListOrganizations returns a list of organizations with pagination
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	query := &adminservice.OrganizationListQuery{
		Search:   c.Query("search"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}

	result, err := h.adminService.ListOrganizations(c.Request.Context(), query)
	if err != nil {
		apierr.InternalError(c, "Failed to list organizations")
		return
	}

	// Convert to response format
	orgs := make([]gin.H, len(result.Data))
	for i, org := range result.Data {
		orgs[i] = organizationResponse(&org)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        orgs,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// GetOrganization returns a single organization
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	org, err := h.adminService.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		if err == adminservice.ErrOrganizationNotFound {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		apierr.InternalError(c, "Failed to get organization")
		return
	}

	// Log view action
	h.logAction(c, admin.AuditActionOrgView, admin.TargetTypeOrganization, orgID, nil, nil)

	c.JSON(http.StatusOK, organizationResponse(org))
}

// GetOrganizationMembers returns the members of an organization
func (h *OrganizationHandler) GetOrganizationMembers(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	org, members, err := h.adminService.GetOrganizationWithMembers(c.Request.Context(), orgID)
	if err != nil {
		if err == adminservice.ErrOrganizationNotFound {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		apierr.InternalError(c, "Failed to get organization members")
		return
	}

	// Convert members to response format
	memberList := make([]gin.H, len(members))
	for i, m := range members {
		memberList[i] = memberResponse(&m)
	}

	c.JSON(http.StatusOK, gin.H{
		"organization": organizationResponse(org),
		"members":      memberList,
	})
}

// DeleteOrganization deletes an organization
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	// Get old data for audit log
	oldOrg, _ := h.adminService.GetOrganization(c.Request.Context(), orgID)

	if err := h.adminService.DeleteOrganization(c.Request.Context(), orgID); err != nil {
		if err == adminservice.ErrOrganizationNotFound {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		if err == adminservice.ErrOrganizationHasActiveRunner {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Cannot delete organization with active runners")
			return
		}
		apierr.InternalError(c, "Failed to delete organization")
		return
	}

	// Log delete action
	h.logAction(c, admin.AuditActionOrgDelete, admin.TargetTypeOrganization, orgID, oldOrg, nil)

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully"})
}

// logAction is a helper method that delegates to the shared LogAdminAction function
func (h *OrganizationHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// organizationResponse creates a sanitized organization response
func organizationResponse(org *organization.Organization) gin.H {
	return gin.H{
		"id":                  org.ID,
		"name":                org.Name,
		"slug":                org.Slug,
		"logo_url":            org.LogoURL,
		"subscription_plan":   org.SubscriptionPlan,
		"subscription_status": org.SubscriptionStatus,
		"created_at":          org.CreatedAt,
		"updated_at":          org.UpdatedAt,
	}
}

// memberResponse creates a sanitized member response
func memberResponse(m *organization.Member) gin.H {
	response := gin.H{
		"id":        m.ID,
		"user_id":   m.UserID,
		"org_id":    m.OrganizationID,
		"role":      m.Role,
		"joined_at": m.JoinedAt,
	}

	if m.User != nil {
		response["user"] = gin.H{
			"id":         m.User.ID,
			"email":      m.User.Email,
			"username":   m.User.Username,
			"name":       m.User.Name,
			"avatar_url": m.User.AvatarURL,
		}
	}

	return response
}
