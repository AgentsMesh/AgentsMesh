package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type APIKeyValidator interface {
	ValidateKey(ctx context.Context, rawKey string) (*APIKeyValidateResult, error)
	UpdateLastUsed(ctx context.Context, id int64) error
}

type APIKeyValidateResult struct {
	APIKeyID       int64
	OrganizationID int64
	CreatedBy      int64
	Scopes         []string
	KeyName        string
}

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

func APIKeyAuthMiddleware(apiKeyValidator APIKeyValidator, orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawKey := extractAPIKey(c)
		if rawKey == "" {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "API key is required")
			return
		}

		result, err := apiKeyValidator.ValidateKey(c.Request.Context(), rawKey)
		if err != nil {
			handleAPIKeyError(c, err)
			c.Abort()
			return
		}

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

		if org.GetID() != result.OrganizationID {
			apierr.AbortForbidden(c, apierr.API_KEY_ORG_MISMATCH, "API key does not belong to this organization")
			return
		}

		tc := &TenantContext{
			OrganizationID:   result.OrganizationID,
			OrganizationSlug: org.GetSlug(),
			UserID:           result.CreatedBy,
			UserRole:         "apikey",
		}
		c.Set("tenant", tc)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		akCtx := &APIKeyContext{
			APIKeyID: result.APIKeyID,
			KeyName:  result.KeyName,
			Scopes:   result.Scopes,
		}
		c.Set("apikey_context", akCtx)
		c.Set("auth_type", "apikey")

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := apiKeyValidator.UpdateLastUsed(ctx, result.APIKeyID); err != nil {
				slog.WarnContext(ctx, "Failed to update API key last_used_at", "key_id", result.APIKeyID, "error", err)
			}
		}()

		c.Next()
	}
}

func RequireScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		akCtxRaw, exists := c.Get("apikey_context")
		if !exists {
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

func GetAPIKeyContext(c *gin.Context) *APIKeyContext {
	if akCtx, exists := c.Get("apikey_context"); exists {
		if ctx, ok := akCtx.(*APIKeyContext); ok {
			return ctx
		}
	}
	return nil
}

func extractAPIKey(c *gin.Context) string {
	if key := c.GetHeader("X-API-Key"); key != "" && strings.HasPrefix(key, "amk_") {
		return key
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" && strings.HasPrefix(parts[1], "amk_") {
			return parts[1]
		}
	}

	return ""
}

func handleAPIKeyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAPIKeyNotFound):
		apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid API key")
	case errors.Is(err, ErrAPIKeyDisabled):
		apierr.Forbidden(c, apierr.API_KEY_DISABLED, "API key is disabled")
	case errors.Is(err, ErrAPIKeyExpired):
		apierr.Unauthorized(c, apierr.API_KEY_EXPIRED, "API key has expired")
	default:
		apierr.InternalError(c, "Failed to validate API key")
	}
}
