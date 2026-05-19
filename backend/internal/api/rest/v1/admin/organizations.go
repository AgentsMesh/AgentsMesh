package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	adminService *adminservice.Service
}

func NewOrganizationHandler(adminSvc *adminservice.Service) *OrganizationHandler {
	return &OrganizationHandler{
		adminService: adminSvc,
	}
}

func (h *OrganizationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	orgsGroup := rg.Group("/organizations")
	{
		orgsGroup.GET("", h.ListOrganizations)
		orgsGroup.GET("/:id", h.GetOrganization)
		orgsGroup.GET("/:id/members", h.GetOrganizationMembers)
		orgsGroup.DELETE("/:id", h.DeleteOrganization)
	}
}

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

func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	org, err := h.adminService.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, adminservice.ErrOrganizationNotFound) {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		apierr.InternalError(c, "Failed to get organization")
		return
	}

	h.logAction(c, admin.AuditActionOrgView, admin.TargetTypeOrganization, orgID, nil, nil)

	c.JSON(http.StatusOK, organizationResponse(org))
}

func (h *OrganizationHandler) GetOrganizationMembers(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	org, members, err := h.adminService.GetOrganizationWithMembers(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, adminservice.ErrOrganizationNotFound) {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		apierr.InternalError(c, "Failed to get organization members")
		return
	}

	memberList := make([]gin.H, len(members))
	for i, m := range members {
		memberList[i] = memberResponse(&m)
	}

	c.JSON(http.StatusOK, gin.H{
		"organization": organizationResponse(org),
		"members":      memberList,
	})
}

func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid organization ID")
		return
	}

	oldOrg, _ := h.adminService.GetOrganization(c.Request.Context(), orgID)

	if err := h.adminService.DeleteOrganization(c.Request.Context(), orgID); err != nil {
		if errors.Is(err, adminservice.ErrOrganizationNotFound) {
			apierr.ResourceNotFound(c, "Organization not found")
			return
		}
		if errors.Is(err, adminservice.ErrOrganizationHasActiveRunner) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Cannot delete organization with active runners")
			return
		}
		apierr.InternalError(c, "Failed to delete organization")
		return
	}

	h.logAction(c, admin.AuditActionOrgDelete, admin.TargetTypeOrganization, orgID, oldOrg, nil)

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted successfully"})
}

func (h *OrganizationHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

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
