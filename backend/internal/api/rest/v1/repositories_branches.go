package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

func (h *RepositoryHandler) SyncBranches(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req SyncBranchesRequest
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
	if !policy.RepositoryPolicy.AllowRead(sub, h.repoResourceWithGrants(
		c.Request.Context(), repoID, repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	branches, err := h.repositoryService.ListBranches(c.Request.Context(), repoID, req.AccessToken)
	if err != nil {
		apierr.InternalError(c, "Failed to sync branches")
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

func (h *RepositoryHandler) ListBranches(c *gin.Context) {
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
	if !policy.RepositoryPolicy.AllowRead(sub, h.repoResourceWithGrants(
		c.Request.Context(), repoID, repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	accessToken := c.Query("access_token")
	if accessToken == "" {
		accessToken = c.GetHeader("X-Git-Access-Token")
	}
	if accessToken == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "Access token required")
		return
	}

	branches, err := h.repositoryService.ListBranches(c.Request.Context(), repoID, accessToken)
	if err != nil {
		apierr.InternalError(c, "Failed to list branches")
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

// Deprecated: use RegisterRepositoryWebhook.
func (h *RepositoryHandler) SetupWebhook(c *gin.Context) {
	h.RegisterRepositoryWebhook(c)
}
