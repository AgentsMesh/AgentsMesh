package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WebhookRouter handles webhook endpoint routing
type WebhookRouter struct {
	db       *gorm.DB
	cfg      *config.Config
	logger   *slog.Logger
	registry *HandlerRegistry
}

// NewWebhookRouter creates a new webhook router
func NewWebhookRouter(db *gorm.DB, cfg *config.Config, logger *slog.Logger) *WebhookRouter {
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	return &WebhookRouter{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		registry: registry,
	}
}

// RegisterRoutes registers webhook routes to the router group
func (r *WebhookRouter) RegisterRoutes(rg *gin.RouterGroup) {
	// GitLab webhook endpoint
	rg.POST("/gitlab", r.handleGitLabWebhook)

	// GitHub webhook endpoint
	rg.POST("/github", r.handleGitHubWebhook)

	// Gitee webhook endpoint
	rg.POST("/gitee", r.handleGiteeWebhook)
}

// handleGitLabWebhook handles GitLab webhook events
func (r *WebhookRouter) handleGitLabWebhook(c *gin.Context) {
	// Verify webhook secret if configured
	if r.cfg.Webhook.GitLabSecret != "" {
		token := c.GetHeader("X-Gitlab-Token")
		if token != r.cfg.Webhook.GitLabSecret {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error":  "invalid webhook token",
			})
			return
		}
	}

	r.processWebhook(c, "gitlab")
}

// handleGitHubWebhook handles GitHub webhook events
func (r *WebhookRouter) handleGitHubWebhook(c *gin.Context) {
	// Verify webhook secret if configured
	if r.cfg.Webhook.GitHubSecret != "" {
		// GitHub uses X-Hub-Signature-256 for HMAC verification
		// For simplicity, we'll use a header token approach similar to GitLab
		// In production, you should use proper HMAC signature verification
		token := c.GetHeader("X-Hub-Signature-256")
		if token == "" {
			token = c.GetHeader("X-GitHub-Token")
		}
		if !r.verifyGitHubSignature(c, r.cfg.Webhook.GitHubSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error":  "invalid webhook signature",
			})
			return
		}
	}

	r.processWebhook(c, "github")
}

// handleGiteeWebhook handles Gitee webhook events
func (r *WebhookRouter) handleGiteeWebhook(c *gin.Context) {
	// Verify webhook secret if configured
	if r.cfg.Webhook.GiteeSecret != "" {
		if !r.verifyGiteeSignature(c, r.cfg.Webhook.GiteeSecret) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status": "error",
				"error":  "invalid webhook signature",
			})
			return
		}
	}

	r.processWebhook(c, "gitee")
}

// processWebhook processes a webhook event from any provider
func (r *WebhookRouter) processWebhook(c *gin.Context, provider string) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		r.logger.Error("failed to parse webhook payload",
			"provider", provider,
			"error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "invalid JSON payload",
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// extractObjectKind extracts the event type from payload based on provider
func (r *WebhookRouter) extractObjectKind(payload map[string]interface{}, provider string, c *gin.Context) string {
	switch provider {
	case "gitlab":
		if kind, ok := payload["object_kind"].(string); ok {
			return kind
		}
	case "github":
		// GitHub uses X-GitHub-Event header
		event := c.GetHeader("X-GitHub-Event")
		if event != "" {
			return r.mapGitHubEventToKind(event)
		}
	case "gitee":
		// Gitee uses X-Gitee-Event header or hook_name in payload
		event := c.GetHeader("X-Gitee-Event")
		if event != "" {
			return r.mapGiteeEventToKind(event)
		}
		if hookName, ok := payload["hook_name"].(string); ok {
			return r.mapGiteeEventToKind(hookName)
		}
	}

	return ""
}

// mapGitHubEventToKind maps GitHub event names to our internal event kinds
func (r *WebhookRouter) mapGitHubEventToKind(event string) string {
	mapping := map[string]string{
		"push":                  "push",
		"pull_request":         "merge_request",
		"check_run":            "pipeline",
		"check_suite":          "pipeline",
		"workflow_run":         "pipeline",
		"status":               "pipeline",
		"issues":               "issue",
		"issue_comment":        "note",
		"pull_request_review":  "note",
	}

	if kind, ok := mapping[event]; ok {
		return kind
	}
	return event
}

// mapGiteeEventToKind maps Gitee event names to our internal event kinds
func (r *WebhookRouter) mapGiteeEventToKind(event string) string {
	mapping := map[string]string{
		"push_hooks":           "push",
		"Push Hook":            "push",
		"merge_request_hooks":  "merge_request",
		"Merge Request Hook":   "merge_request",
		"issue_hooks":          "issue",
		"Issue Hook":           "issue",
		"note_hooks":           "note",
		"Note Hook":            "note",
	}

	if kind, ok := mapping[event]; ok {
		return kind
	}
	return event
}

// verifyGitHubSignature verifies GitHub webhook signature using HMAC-SHA256
func (r *WebhookRouter) verifyGitHubSignature(c *gin.Context, secret string) bool {
	// Get the signature from header
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		// Fallback: check custom token header for backwards compatibility
		token := c.GetHeader("X-GitHub-Token")
		return token != "" && token == secret
	}

	// Signature format: sha256=<hex_signature>
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedMAC := signature[7:] // Remove "sha256=" prefix

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		r.logger.Error("failed to read request body for signature verification", "error", err)
		return false
	}
	// Restore the body for later processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Calculate HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actualMAC := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(expectedMAC), []byte(actualMAC))
}

// verifyGiteeSignature verifies Gitee webhook signature using HMAC-SHA256
func (r *WebhookRouter) verifyGiteeSignature(c *gin.Context, secret string) bool {
	// Gitee supports multiple signature methods
	// Method 1: X-Gitee-Token (simple token comparison)
	token := c.GetHeader("X-Gitee-Token")
	if token != "" {
		return token == secret
	}

	// Method 2: X-Gitee-Timestamp + X-Gitee-Token with HMAC
	timestamp := c.GetHeader("X-Gitee-Timestamp")
	signature := c.GetHeader("X-Gitee-Token")
	if timestamp != "" && signature != "" {
		// Read the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			r.logger.Error("failed to read request body for Gitee signature verification", "error", err)
			return false
		}
		// Restore the body for later processing
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Calculate HMAC-SHA256: timestamp + "\n" + body
		stringToSign := timestamp + "\n" + string(body)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		return hmac.Equal([]byte(signature), []byte(expectedMAC))
	}

	return false
}
