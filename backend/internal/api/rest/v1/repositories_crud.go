package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

// ListRepositories lists configured repositories visible to the requesting user
// GET /api/v1/organizations/:slug/repositories
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	filter := policy.RepositoryPolicy.ListFilter(sub)
	repos, err := h.repositoryService.ListByOrganizationForUser(c.Request.Context(), tenant.OrganizationID, filter.VisibilityUserID)
	if err != nil {
		apierr.InternalError(c, "Failed to list repositories")
		return
	}

	c.JSON(http.StatusOK, gin.H{"repositories": repos})
}

// GetRepository returns repository by ID
// GET /api/v1/organizations/:slug/repositories/:id
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"repository": repo})
}
