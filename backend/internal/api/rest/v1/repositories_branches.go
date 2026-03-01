package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SyncBranches syncs branches from git provider
// POST /api/v1/organizations/:slug/repositories/:id/sync-branches
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

	tenant := middleware.GetTenant(c)

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
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

// ListBranches lists repository branches
// GET /api/v1/organizations/:slug/repositories/:id/branches
func (h *RepositoryHandler) ListBranches(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Get access token from query or header
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

// SetupWebhook sets up webhook for repository
// POST /api/v1/organizations/:slug/repositories/:id/webhook
// Deprecated: Use RegisterRepositoryWebhook instead
func (h *RepositoryHandler) SetupWebhook(c *gin.Context) {
	// Delegate to the new RegisterRepositoryWebhook handler
	h.RegisterRepositoryWebhook(c)
}
