package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

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

func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	var req CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}

	// Skip quota check for re-imports.
	if h.billingService != nil {
		_, existsErr := h.repositoryService.GetBySlug(
			c.Request.Context(), tenant.OrganizationID,
			req.ProviderType, req.ProviderBaseURL, req.Slug,
		)
		isNewRepo := existsErr == repository.ErrRepositoryNotFound
		if isNewRepo {
			if err := h.billingService.CheckQuota(c.Request.Context(), tenant.OrganizationID, "repositories", 1); err != nil {
				if errors.Is(err, billing.ErrQuotaExceeded) {
					apierr.PaymentRequired(c, apierr.REPOSITORY_QUOTA_EXCEEDED, "Repository quota exceeded. Please upgrade your plan to add more repositories.")
					return
				}
				if errors.Is(err, billing.ErrSubscriptionFrozen) {
					apierr.PaymentRequired(c, apierr.SUBSCRIPTION_FROZEN, "Your subscription has expired. Please renew to continue.")
					return
				}
				apierr.InternalError(c, "Failed to check quota")
				return
			}
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
		HttpCloneURL:     req.HttpCloneURL,
		SshCloneURL:      req.SshCloneURL,
		ExternalID:       req.ExternalID,
		Name:             req.Name,
		Slug:             req.Slug,
		DefaultBranch:    defaultBranch,
		TicketPrefix:     ticketPrefix,
		Visibility:       visibility,
		ImportedByUserID: &userID,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to create repository")
		return
	}

	c.JSON(http.StatusOK, gin.H{"repository": repo})
}

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

func (h *RepositoryHandler) UpdateRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req UpdateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
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
	if req.HttpCloneURL != nil {
		updates["http_clone_url"] = *req.HttpCloneURL
	}
	if req.SshCloneURL != nil {
		updates["ssh_clone_url"] = *req.SshCloneURL
	}

	repo, err = h.repositoryService.Update(c.Request.Context(), repoID, updates)
	if err != nil {
		apierr.InternalError(c, "Failed to update repository")
		return
	}

	c.JSON(http.StatusOK, gin.H{"repository": repo})
}

func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	if !policy.AllowAdmin(sub, tenant.OrganizationID) {
		apierr.ForbiddenAdmin(c)
		return
	}

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	if !policy.RepositoryPolicy.AllowWrite(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	if err := h.repositoryService.Delete(c.Request.Context(), repoID); err != nil {
		apierr.InternalError(c, "Failed to delete repository")
		return
	}

	if h.grantService != nil {
		_ = h.grantService.CleanupByResource(c.Request.Context(), grant.TypeRepository, grant.IntResourceID(repoID))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository deleted"})
}
