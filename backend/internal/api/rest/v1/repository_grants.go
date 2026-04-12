package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

func (h *RepositoryHandler) ListRepositoryGrants(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	if _, err := h.repositoryService.GetByID(c.Request.Context(), repoID); err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}

	grants, err := h.grantService.ListGrants(c.Request.Context(), grant.TypeRepository, grant.IntResourceID(repoID))
	if err != nil {
		apierr.InternalError(c, "Failed to list grants")
		return
	}
	c.JSON(http.StatusOK, gin.H{"grants": grants})
}

func (h *RepositoryHandler) GrantRepositoryAccess(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req grantAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}
	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	g, err := h.grantService.GrantAccess(c.Request.Context(),
		tenant.OrganizationID, grant.TypeRepository, grant.IntResourceID(repoID), req.UserID, tenant.UserID)
	if err != nil {
		handleGrantError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"grant": g})
}

func (h *RepositoryHandler) RevokeRepositoryGrant(c *gin.Context) {
	if h.grantService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Grant service not configured")
		return
	}
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}
	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	grantID, err := strconv.ParseInt(c.Param("grant_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid grant ID")
		return
	}

	if err := h.grantService.RevokeAccess(c.Request.Context(), grant.TypeRepository, grant.IntResourceID(repoID), grantID); err != nil {
		apierr.ResourceNotFound(c, "Grant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Grant revoked"})
}
