package middleware

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type OrganizationGetter interface {
	GetID() int64
	GetSlug() string
	GetName() string
}

type OrganizationService interface {
	GetBySlug(ctx context.Context, slug string) (OrganizationGetter, error)
	IsMember(ctx context.Context, orgID, userID int64) (bool, error)
	GetMemberRole(ctx context.Context, orgID, userID int64) (string, error)
}

type TenantContext struct {
	OrganizationID   int64
	OrganizationSlug string
	UserID           int64
	UserRole         string // 'owner', 'admin', 'member'
	PodID *int64
}

type tenantContextKey struct{}

func GetTenant(c interface{}) *TenantContext {
	switch ctx := c.(type) {
	case *gin.Context:
		if tc, exists := ctx.Get("tenant"); exists {
			if tenant, ok := tc.(*TenantContext); ok {
				return tenant
			}
		}
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

func GetUserID(c *gin.Context) int64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int64); ok {
			return id
		}
	}
	return 0
}

func SetTenant(ctx context.Context, tc *TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tc)
}

func TenantMiddleware(orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgSlug := c.Param("slug")
		if orgSlug == "" {
			apierr.AbortBadRequest(c, apierr.VALIDATION_FAILED, "Organization slug is required")
			return
		}

		userID := GetUserID(c)
		if userID == 0 {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "User not authenticated")
			return
		}

		org, err := orgService.GetBySlug(c.Request.Context(), orgSlug)
		if err != nil {
			apierr.AbortNotFound(c, apierr.RESOURCE_NOT_FOUND, "Organization not found")
			return
		}

		isMember, err := orgService.IsMember(c.Request.Context(), org.GetID(), userID)
		if err != nil || !isMember {
			apierr.AbortForbidden(c, apierr.NOT_ORG_MEMBER, "You are not a member of this organization")
			return
		}

		role, err := orgService.GetMemberRole(c.Request.Context(), org.GetID(), userID)
		if err != nil {
			role = "member"
		}

		tc := &TenantContext{
			OrganizationID:   org.GetID(),
			OrganizationSlug: org.GetSlug(),
			UserID:           userID,
			UserRole:         role,
		}

		c.Set("tenant", tc)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tc := GetTenant(c)
		if tc == nil {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
			return
		}

		hasRole := false
		for _, role := range roles {
			if tc.UserRole == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			apierr.AbortForbidden(c, apierr.INSUFFICIENT_PERMISSIONS, "Insufficient permissions")
			return
		}

		c.Next()
	}
}

func RequireOwner() gin.HandlerFunc {
	return RequireRole("owner")
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole("owner", "admin")
}
