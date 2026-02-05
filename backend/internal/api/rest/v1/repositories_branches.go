package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SyncBranches syncs branches from git provider
// POST /api/v1/organizations/:slug/repositories/:id/sync-branches
func (h *RepositoryHandler) SyncBranches(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	var req SyncBranchesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	branches, err := h.repositoryService.ListBranches(c.Request.Context(), repoID, req.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync branches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

// ListBranches lists repository branches
// GET /api/v1/organizations/:slug/repositories/:id/branches
func (h *RepositoryHandler) ListBranches(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get access token from query or header
	accessToken := c.Query("access_token")
	if accessToken == "" {
		accessToken = c.GetHeader("X-Git-Access-Token")
	}
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Access token required"})
		return
	}

	branches, err := h.repositoryService.ListBranches(c.Request.Context(), repoID, accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list branches"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

// SetupWebhook sets up webhook for repository
// POST /api/v1/organizations/:slug/repositories/:id/webhook
func (h *RepositoryHandler) SetupWebhook(c *gin.Context) {
	// TODO: Implement webhook setup
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Webhook setup not implemented yet",
	})
}
