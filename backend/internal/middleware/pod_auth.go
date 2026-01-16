package middleware

import (
	"context"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/gin-gonic/gin"
)

// PodService interface for pod lookup
type PodService interface {
	GetPodByKey(ctx context.Context, podKey string) (*agentpod.Pod, error)
}

// PodAuthMiddleware extracts pod key from X-Pod-Key header
// and sets up the tenant context based on the pod's organization.
//
// Routes: /api/v1/orgs/:slug/pod/*
// The middleware validates that the URL's org slug matches the pod's organization.
func PodAuthMiddleware(podService PodService, orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		podKey := c.GetHeader("X-Pod-Key")
		if podKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "X-Pod-Key header required",
			})
			c.Abort()
			return
		}

		// Get pod by key
		pod, err := podService.GetPodByKey(c.Request.Context(), podKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid pod key",
			})
			c.Abort()
			return
		}

		// Get org slug from URL path parameter
		orgSlug := c.Param("slug")
		if orgSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Organization slug is required in URL path",
			})
			c.Abort()
			return
		}

		// Lookup organization by slug
		org, err := orgService.GetBySlug(c.Request.Context(), orgSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Organization not found",
			})
			c.Abort()
			return
		}

		// Verify pod belongs to this organization
		if pod.OrganizationID != org.GetID() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Pod does not belong to this organization",
			})
			c.Abort()
			return
		}

		// Create tenant context with pod info
		// Use pod's CreatedByID as the user ID for permission checks
		// This ensures MCP tools operate with the pod creator's permissions
		tc := &TenantContext{
			OrganizationID:   org.GetID(),
			OrganizationSlug: org.GetSlug(),
			UserID:           pod.CreatedByID, // Use pod creator's ID
			UserRole:         "pod",           // Special role for pod-based access
		}

		// Store pod key in context for later use
		c.Set("pod_key", podKey)
		c.Set("pod", pod)
		c.Set("tenant", tc)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
