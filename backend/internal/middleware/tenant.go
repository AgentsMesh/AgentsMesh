package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// OrganizationGetter interface for fetching organization info
type OrganizationGetter interface {
	GetID() int64
	GetSlug() string
	GetName() string
}

// OrganizationService interface for organization lookup
type OrganizationService interface {
	GetBySlug(ctx context.Context, slug string) (OrganizationGetter, error)
	IsMember(ctx context.Context, orgID, userID int64) (bool, error)
	GetMemberRole(ctx context.Context, orgID, userID int64) (string, error)
}

// TenantContext holds tenant information for the current request
type TenantContext struct {
	OrganizationID   int64
	OrganizationSlug string
	UserID           int64
	UserRole         string // 'owner', 'admin', 'member'
}

type tenantContextKey struct{}

// GetTenant retrieves the tenant context from gin.Context or request context
func GetTenant(c interface{}) *TenantContext {
	switch ctx := c.(type) {
	case *gin.Context:
		// First try gin context
		if tc, exists := ctx.Get("tenant"); exists {
			if tenant, ok := tc.(*TenantContext); ok {
				return tenant
			}
		}
		// Then try request context
		if tc, ok := ctx.Request.Context().Value(tenantContextKey{}).(*TenantContext); ok {
			return tc
		}
	case context.Context:
		if tc, ok := ctx.Value(tenantContextKey{}).(*TenantContext); ok {
			return tc
		}
	}
	return nil
}

// GetUserID retrieves the user ID from gin.Context
func GetUserID(c *gin.Context) int64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
	}
	return 0
}

// SetTenant sets the tenant context in the request context
func SetTenant(ctx context.Context, tc *TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tc)
}

// TenantMiddleware extracts tenant information from the URL path parameter
// and validates that the user is a member of the organization
func TenantMiddleware(orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get org slug from URL path parameter (e.g., /orgs/:slug/...)
		orgSlug := c.Param("slug")
		if orgSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Organization slug is required",
			})
			c.Abort()
			return
		}

		// Get user ID from auth middleware
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		// Lookup organization
		org, err := orgService.GetBySlug(c.Request.Context(), orgSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Organization not found",
			})
			c.Abort()
			return
		}

		// Check membership
		isMember, err := orgService.IsMember(c.Request.Context(), org.GetID(), userID)
		if err != nil || !isMember {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You are not a member of this organization",
			})
			c.Abort()
			return
		}

		// Get user role in organization
		role, err := orgService.GetMemberRole(c.Request.Context(), org.GetID(), userID)
		if err != nil {
			role = "member" // Default to member if role lookup fails
		}

		// Create tenant context
		tc := &TenantContext{
			OrganizationID:   org.GetID(),
			OrganizationSlug: org.GetSlug(),
			UserID:           userID,
			UserRole:         role,
		}

		// Set tenant context in both gin context and request context
		c.Set("tenant", tc)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireRole middleware checks if the user has the required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenant(c)
		if tc == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Tenant context not found",
			})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range roles {
			if tc.UserRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOwner is a convenience middleware for requiring owner role
func RequireOwner() gin.HandlerFunc {
	return RequireRole("owner")
}

// RequireAdmin is a convenience middleware for requiring admin or owner role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("owner", "admin")
}
