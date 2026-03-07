package webhooks

import (
	"crypto/subtle"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// handleGitLabWebhook handles GitLab webhook events (global endpoint)
func (r *WebhookRouter) handleGitLabWebhook(c *gin.Context) {
	if r.cfg.Webhook.GitLabSecret == "" {
		r.logger.Warn("GitLab webhook received but secret is not configured")
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
		return
	}
	token := c.GetHeader("X-Gitlab-Token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(r.cfg.Webhook.GitLabSecret)) != 1 {
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook token")
		return
	}

	r.processWebhook(c, "gitlab")
}

// handleGitHubWebhook handles GitHub webhook events (global endpoint)
func (r *WebhookRouter) handleGitHubWebhook(c *gin.Context) {
	if r.cfg.Webhook.GitHubSecret == "" {
		r.logger.Warn("GitHub webhook received but secret is not configured")
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
		return
	}
	if !r.verifyGitHubSignature(c, r.cfg.Webhook.GitHubSecret) {
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook signature")
		return
	}

	r.processWebhook(c, "github")
}

// handleGiteeWebhook handles Gitee webhook events (global endpoint)
func (r *WebhookRouter) handleGiteeWebhook(c *gin.Context) {
	if r.cfg.Webhook.GiteeSecret == "" {
		r.logger.Warn("Gitee webhook received but secret is not configured")
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "webhook secret not configured")
		return
	}
	if !r.verifyGiteeSignature(c, r.cfg.Webhook.GiteeSecret) {
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "invalid webhook signature")
		return
	}

	r.processWebhook(c, "gitee")
}

// processWebhook processes a webhook event from any provider (global endpoint)
func (r *WebhookRouter) processWebhook(c *gin.Context, provider string) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		r.logger.Error("failed to parse webhook payload",
			"provider", provider,
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

	r.logger.Info("received webhook",
		"provider", provider,
		"object_kind", objectKind)

	// Create webhook context
	ctx := NewWebhookContext(c.Request.Context(), r.db, payload)

	// Override object kind if extracted differently
	if ctx.ObjectKind == "" {
		ctx.ObjectKind = objectKind
	}

	// Process the webhook
	result, err := r.registry.Process(ctx)
	if err != nil {
		r.logger.Error("webhook processing failed",
			"provider", provider,
			"object_kind", objectKind,
			"error", err)
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}
