package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/repository"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

// RegisterRepositoryWebhook registers a webhook for a repository
// POST /api/v1/organizations/:slug/repositories/:id/webhook
func (h *RepositoryHandler) RegisterRepositoryWebhook(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)
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

	webhookService := h.repositoryService.GetWebhookService()
	if webhookService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Webhook service not available")
		return
	}

	result, err := webhookService.RegisterWebhookForRepository(c.Request.Context(), repo, tenant.OrganizationSlug, userID)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// DeleteRepositoryWebhook deletes a webhook from a repository
// DELETE /api/v1/organizations/:slug/repositories/:id/webhook
func (h *RepositoryHandler) DeleteRepositoryWebhook(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	userID := middleware.GetUserID(c)
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

	webhookService := h.repositoryService.GetWebhookService()
	if webhookService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Webhook service not available")
		return
	}

	if err := webhookService.DeleteWebhookForRepository(c.Request.Context(), repo, userID); err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook deleted"})
}

// GetRepositoryWebhookStatus returns the webhook status for a repository
// GET /api/v1/organizations/:slug/repositories/:id/webhook/status
func (h *RepositoryHandler) GetRepositoryWebhookStatus(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	repo, err := h.repositoryService.GetByID(c.Request.Context(), repoID)
	if err != nil {
		apierr.ResourceNotFound(c, "Repository not found")
		return
	}

	if !policy.RepositoryPolicy.AllowRead(sub, policy.VisibleResource(
		repo.OrganizationID, repo.ImportedByUserID, repo.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	webhookService := h.repositoryService.GetWebhookService()
	if webhookService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Webhook service not available")
		return
	}

	status := webhookService.GetWebhookStatus(c.Request.Context(), repo)
	c.JSON(http.StatusOK, gin.H{"webhook_status": status})
}

// GetRepositoryWebhookSecret returns the webhook secret for manual configuration
// GET /api/v1/organizations/:slug/repositories/:id/webhook/secret
func (h *RepositoryHandler) GetRepositoryWebhookSecret(c *gin.Context) {
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

	webhookService := h.repositoryService.GetWebhookService()
	if webhookService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Webhook service not available")
		return
	}

	secret, err := webhookService.GetWebhookSecret(c.Request.Context(), repo)
	if err != nil {
		if err == repository.ErrWebhookNotFound {
			apierr.ResourceNotFound(c, "Webhook not configured")
			return
		}
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"webhook_url":    repo.WebhookConfig.URL,
		"webhook_secret": secret,
		"events":         repo.WebhookConfig.Events,
	})
}

// MarkRepositoryWebhookConfigured marks a webhook as manually configured
// POST /api/v1/organizations/:slug/repositories/:id/webhook/configured
func (h *RepositoryHandler) MarkRepositoryWebhookConfigured(c *gin.Context) {
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

	webhookService := h.repositoryService.GetWebhookService()
	if webhookService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Webhook service not available")
		return
	}

	if err := webhookService.MarkWebhookAsConfigured(c.Request.Context(), repo); err != nil {
		if err == repository.ErrWebhookNotFound {
			apierr.ResourceNotFound(c, "Webhook not configured")
			return
		}
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook marked as configured"})
}
