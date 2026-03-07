package webhooks

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// handleGitLabWebhookWithRepo handles GitLab webhook events with repo_id
// POST /webhooks/:org_slug/gitlab/:repo_id
func (r *WebhookRouter) handleGitLabWebhookWithRepo(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repo_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}
	orgSlug := c.Param("org_slug")

	// Verify webhook secret using repository-specific secret
	if r.webhookService != nil {
		token := c.GetHeader("X-Gitlab-Token")
		valid, err := r.webhookService.VerifyWebhookSecret(c.Request.Context(), repoID, token)
		if err != nil || !valid {
			// Fallback to global secret (constant-time comparison)
			if r.cfg.Webhook.GitLabSecret == "" || subtle.ConstantTimeCompare([]byte(token), []byte(r.cfg.Webhook.GitLabSecret)) != 1 {
				apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook token")
				return
			}
		}
	} else if r.cfg.Webhook.GitLabSecret != "" {
		token := c.GetHeader("X-Gitlab-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(r.cfg.Webhook.GitLabSecret)) != 1 {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook token")
			return
		}
	} else {
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
		return
	}

	r.processWebhookWithRepo(c, "gitlab", orgSlug, repoID)
}

// handleGitHubWebhookWithRepo handles GitHub webhook events with repo_id
// POST /webhooks/:org_slug/github/:repo_id
func (r *WebhookRouter) handleGitHubWebhookWithRepo(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repo_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}
	orgSlug := c.Param("org_slug")

	// Verify webhook secret using repository-specific secret
	verified := false
	if r.webhookService != nil {
		// For GitHub, we need to verify HMAC signature with repo-specific secret
		repo, err := r.webhookService.GetRepositoryByIDWithWebhook(c.Request.Context(), repoID)
		if err == nil && repo.WebhookConfig != nil && repo.WebhookConfig.Secret != "" {
			if r.verifyGitHubSignature(c, repo.WebhookConfig.Secret) {
				verified = true
			}
		}
	}

	// Fallback to global secret if repo-specific verification failed
	if !verified {
		if r.cfg.Webhook.GitHubSecret == "" {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
			return
		}
		if !r.verifyGitHubSignature(c, r.cfg.Webhook.GitHubSecret) {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook signature")
			return
		}
	}

	r.processWebhookWithRepo(c, "github", orgSlug, repoID)
}

// handleGiteeWebhookWithRepo handles Gitee webhook events with repo_id
// POST /webhooks/:org_slug/gitee/:repo_id
func (r *WebhookRouter) handleGiteeWebhookWithRepo(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repo_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}
	orgSlug := c.Param("org_slug")

	// Verify webhook secret using repository-specific secret
	verified := false
	if r.webhookService != nil {
		repo, err := r.webhookService.GetRepositoryByIDWithWebhook(c.Request.Context(), repoID)
		if err == nil && repo.WebhookConfig != nil && repo.WebhookConfig.Secret != "" {
			if r.verifyGiteeSignature(c, repo.WebhookConfig.Secret) {
				verified = true
			}
		}
	}

	// Fallback to global secret if repo-specific verification failed
	if !verified {
		if r.cfg.Webhook.GiteeSecret == "" {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
			return
		}
		if !r.verifyGiteeSignature(c, r.cfg.Webhook.GiteeSecret) {
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook signature")
			return
		}
	}

	r.processWebhookWithRepo(c, "gitee", orgSlug, repoID)
}

// processWebhookWithRepo processes a webhook event with org_slug and repo_id context
func (r *WebhookRouter) processWebhookWithRepo(c *gin.Context, provider, orgSlug string, repoID int64) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		r.logger.Error("failed to parse webhook payload",
			"provider", provider,
			"repo_id", repoID,
			"error", err)
		apierr.BadRequest(c, apierr.INVALID_INPUT, "invalid JSON payload")
		return
	}

	// Determine object kind based on provider
	objectKind := r.extractObjectKind(payload, provider, c)

	// GitLab legacy compatibility: build -> job
	if objectKind == "build" {
		objectKind = "job"
	}

	r.logger.Info("received webhook with repo context",
		"provider", provider,
		"object_kind", objectKind,
		"org_slug", orgSlug,
		"repo_id", repoID)

	// Create webhook context with repo info
	ctx := NewWebhookContext(c.Request.Context(), r.db, payload)
	ctx.OrgSlug = orgSlug
	ctx.RepoID = repoID

	// Override object kind if extracted differently
	if ctx.ObjectKind == "" {
		ctx.ObjectKind = objectKind
	}

	// Get repository to set OrganizationID
	if r.repoService != nil {
		repo, err := r.repoService.GetByID(c.Request.Context(), repoID)
		if err != nil {
			r.logger.Error("repository not found for webhook",
				"repo_id", repoID,
				"error", err)
			apierr.ResourceNotFound(c, fmt.Sprintf("repository not found: %d", repoID))
			return
		}
		ctx.OrganizationID = repo.OrganizationID

		// Verify project_id in payload matches repository's external_id
		if ctx.ProjectID != "" && ctx.ProjectID != repo.ExternalID {
			r.logger.Warn("project_id mismatch in webhook",
				"expected", repo.ExternalID,
				"received", ctx.ProjectID,
				"repo_id", repoID)
			// Continue processing but log the warning
		}
	}

	// Process MR and Pipeline events with full context
	if (objectKind == "merge_request" || objectKind == "pipeline") && r.mrSyncService != nil && r.eventBus != nil {
		result, err := r.processMROrPipelineEvent(ctx, objectKind)
		if err != nil {
			r.logger.Error("MR/Pipeline event processing failed",
				"object_kind", objectKind,
				"repo_id", repoID,
				"error", err)
			// Don't fail the webhook - just log and return success
			c.JSON(http.StatusOK, gin.H{
				"status":  "partial",
				"error":   err.Error(),
				"handler": objectKind,
			})
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Process other events using registry
	result, err := r.registry.Process(ctx)
	if err != nil {
		r.logger.Error("webhook processing failed",
			"provider", provider,
			"object_kind", objectKind,
			"repo_id", repoID,
			"error", err)
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}
