package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// APIKeyValidator interface for validating API keys (decoupled from service)
type APIKeyValidator interface {
	ValidateKey(ctx context.Context, rawKey string) (*APIKeyValidateResult, error)
	UpdateLastUsed(ctx context.Context, id int64) error
}

// APIKeyValidateResult holds the validation result
type APIKeyValidateResult struct {
	APIKeyID       int64
	OrganizationID int64
	CreatedBy      int64
	Scopes         []string
	KeyName        string
}

// APIKeyContext stores API key authentication context
type APIKeyContext struct {
	APIKeyID int64
	KeyName  string
	Scopes   []string
}

// APIKeyError sentinel errors used by the middleware for error matching.
// These mirror service-layer errors and are set by the MiddlewareAdapter.
var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyDisabled = errors.New("api key is disabled")
	ErrAPIKeyExpired  = errors.New("api key has expired")
)

// APIKeyAuthMiddleware validates API key and sets TenantContext for downstream handlers.
// Supports two header formats:
//   - X-API-Key: amk_...
//   - Authorization: Bearer amk_...
func APIKeyAuthMiddleware(apiKeyValidator APIKeyValidator, orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "API key is required")
			return
		}

		// Validate key
		result, err := apiKeyValidator.ValidateKey(c.Request.Context(), rawKey)
		if err != nil {
			handleAPIKeyError(c, err)
			c.Abort()
			return
		}

		// Resolve organization from :slug path parameter
		orgSlug := c.Param("slug")
		if orgSlug == "" {
			apierr.AbortBadRequest(c, apierr.VALIDATION_FAILED, "Organization slug is required")
			return
		}

		org, err := orgService.GetBySlug(c.Request.Context(), orgSlug)
		if err != nil {
			apierr.AbortNotFound(c, apierr.RESOURCE_NOT_FOUND, "Organization not found")
			return
		}

		// Verify key belongs to the requested organization
		if org.GetID() != result.OrganizationID {
			apierr.AbortForbidden(c, apierr.API_KEY_ORG_MISMATCH, "API key does not belong to this organization")
			return
		}

		// Construct TenantContext compatible with existing handlers.
		// UserID = API key creator, UserRole = "apikey" (passes existing role checks)
		tc := &TenantContext{
			OrganizationID:   result.OrganizationID,
			OrganizationSlug: org.GetSlug(),
			UserID:           result.CreatedBy,
			UserRole:         "apikey",
		}
		c.Set("tenant", tc)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		// Set API key context for scope checking
		akCtx := &APIKeyContext{
			APIKeyID: result.APIKeyID,
			KeyName:  result.KeyName,
			Scopes:   result.Scopes,
		}
		c.Set("apikey_context", akCtx)
		c.Set("auth_type", "apikey")

		// Update last_used_at asynchronously (fire-and-forget with timeout)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := apiKeyValidator.UpdateLastUsed(ctx, result.APIKeyID); err != nil {
				slog.Warn("Failed to update API key last_used_at", "key_id", result.APIKeyID, "error", err)
			}
		}()

		c.Next()
	}
}

// RequireScope checks that the API key has at least one of the required scopes.
// If no APIKeyContext is present (user auth), it passes through.
func RequireScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		akCtxRaw, exists := c.Get("apikey_context")
		if !exists {
			// Not API key auth (user auth), pass through
			c.Next()
			return
		}

		akCtx, ok := akCtxRaw.(*APIKeyContext)
		if !ok {
			c.AbortWithStatusJSON(500, apierr.ErrorResponse{
				Error: "Invalid API key context",
				Code:  apierr.INTERNAL_ERROR,
			})
			return
		}

		for _, required := range scopes {
			for _, granted := range akCtx.Scopes {
				if granted == required {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(403, gin.H{
			"error":           "Insufficient scope",
			"code":            apierr.INSUFFICIENT_SCOPE,
			"required_scopes": scopes,
		})
	}
}

// GetAPIKeyContext retrieves the API key context from gin.Context
func GetAPIKeyContext(c *gin.Context) *APIKeyContext {
	if akCtx, exists := c.Get("apikey_context"); exists {
		if ctx, ok := akCtx.(*APIKeyContext); ok {
			return ctx
		}
	}
	return nil
}

// extractAPIKey extracts the API key from request headers
func extractAPIKey(c *gin.Context) string {
	// Priority 1: X-API-Key header (must start with "amk_" prefix)
	if key := c.GetHeader("X-API-Key"); key != "" && strings.HasPrefix(key, "amk_") {
		return key
	}

	// Priority 2: Authorization: Bearer amk_...
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" && strings.HasPrefix(parts[1], "amk_") {
			return parts[1]
		}
	}

	return ""
}

// handleAPIKeyError maps service errors to HTTP responses using errors.Is()
func handleAPIKeyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAPIKeyNotFound):
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid API key")
	case errors.Is(err, ErrAPIKeyDisabled):
		apierr.Forbidden(c, apierr.API_KEY_DISABLED, "API key is disabled")
	case errors.Is(err, ErrAPIKeyExpired):
		apierr.Unauthorized(c, apierr.TOKEN_EXPIRED, "API key has expired")
	default:
		apierr.InternalError(c, "Failed to validate API key")
	}
}
