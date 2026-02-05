package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/gin-gonic/gin"
)

// ListRepositories lists configured repositories
// GET /api/v1/organizations/:slug/repositories
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	repos, err := h.repositoryService.ListByOrganization(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list repositories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"repositories": repos})
}

// CreateRepository creates a new repository configuration
// POST /api/v1/organizations/:slug/repositories
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	// Check repository quota before creation
	if h.billingService != nil {
		if err := h.billingService.CheckQuota(c.Request.Context(), tenant.OrganizationID, "repositories", 1); err != nil {
			if err == billing.ErrQuotaExceeded {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Repository quota exceeded. Please upgrade your plan to add more repositories.",
					"code":  "REPOSITORY_QUOTA_EXCEEDED",
				})
				return
			}
			if err == billing.ErrSubscriptionFrozen {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Your subscription has expired. Please renew to continue.",
					"code":  "SUBSCRIPTION_FROZEN",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			return
		}
	}

	defaultBranch := req.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "organization"
	}

	var ticketPrefix *string
	if req.TicketPrefix != "" {
		ticketPrefix = &req.TicketPrefix
	}

	repo, err := h.repositoryService.Create(c.Request.Context(), &repository.CreateRequest{
		OrganizationID:   tenant.OrganizationID,
		ProviderType:     req.ProviderType,
		ProviderBaseURL:  req.ProviderBaseURL,
		CloneURL:         req.CloneURL,
		ExternalID:       req.ExternalID,
		Name:             req.Name,
		FullPath:         req.FullPath,
		DefaultBranch:    defaultBranch,
		TicketPrefix:     ticketPrefix,
		Visibility:       visibility,
		ImportedByUserID: &userID,
	})
	if err != nil {
		if err == repository.ErrRepositoryExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Repository already configured"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create repository"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"repository": repo})
}

// GetRepository returns repository by ID
// GET /api/v1/organizations/:slug/repositories/:id
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if repo.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"repository": repo})
}

// UpdateRepository updates a repository configuration
// PUT /api/v1/organizations/:slug/repositories/:id
func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	var req UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.DefaultBranch != "" {
		updates["default_branch"] = req.DefaultBranch
	}
	if req.TicketPrefix != "" {
		updates["ticket_prefix"] = req.TicketPrefix
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	repo, err = h.repositoryService.Update(c.Request.Context(), repoID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update repository"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"repository": repo})
}

// DeleteRepository deletes a repository configuration
// DELETE /api/v1/organizations/:slug/repositories/:id
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	if repo.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.repositoryService.Delete(c.Request.Context(), repoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete repository"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository deleted"})
}
