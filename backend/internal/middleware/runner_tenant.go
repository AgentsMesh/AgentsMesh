package middleware

import (
	"context"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/gin-gonic/gin"
)

// RunnerService interface for runner authentication
type RunnerService interface {
	ValidateRunnerAuth(ctx context.Context, nodeID, authToken string) (*runner.Runner, error)
}

// RunnerTenantMiddleware validates runner authentication using auth_token
// and sets tenant context. Used for runner-specific routes that don't use JWT.
//
// Authentication is done via:
//   - Query parameters: ?node_id=xxx&token=xxx
//   - Or Authorization header: Bearer <token> (with node_id in query)
//
// Routes using this middleware:
//   - POST /api/v1/orgs/:slug/runners/heartbeat
//   - GET  /api/v1/orgs/:slug/ws/runners
func RunnerTenantMiddleware(runnerService RunnerService, orgService OrganizationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get org slug from URL path parameter
		orgSlug := c.Param("slug")
		if orgSlug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Organization slug is required"})
			c.Abort()
			return
		}

		// Get runner authentication info
		nodeID := c.Query("node_id")
		authToken := c.Query("token")

		// Fall back to Authorization header for token
		if authToken == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				authToken = authHeader[7:]
			}
		}

		if nodeID == "" || authToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Runner authentication required (node_id and token)"})
			c.Abort()
			return
		}

		// Validate runner authentication
		r, err := runnerService.ValidateRunnerAuth(c.Request.Context(), nodeID, authToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid runner authentication"})
			c.Abort()
			return
		}

		// Lookup organization by slug
		org, err := orgService.GetBySlug(c.Request.Context(), orgSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
			c.Abort()
			return
		}

		// Verify runner belongs to this organization
		if r.OrganizationID != org.GetID() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Runner does not belong to this organization"})
			c.Abort()
			return
		}

		// Create tenant context (reusing TenantContext structure)
		tc := &TenantContext{
			OrganizationID:   org.GetID(),
			OrganizationSlug: org.GetSlug(),
			UserID:           0,        // Runner has no user
			UserRole:         "runner", // Special role for runners
		}

		// Set contexts
		c.Set("tenant", tc)
		c.Set("runner_id", r.ID)
		c.Set("runner", r)
		ctx := SetTenant(c.Request.Context(), tc)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetRunnerID retrieves the runner ID from gin.Context
func GetRunnerID(c *gin.Context) int64 {
	if runnerID, exists := c.Get("runner_id"); exists {
		if id, ok := runnerID.(int64); ok {
			return id
		}
	}
	return 0
}

// GetRunner retrieves the runner from gin.Context
func GetRunner(c *gin.Context) *runner.Runner {
	if r, exists := c.Get("runner"); exists {
		if runner, ok := r.(*runner.Runner); ok {
			return runner
		}
	}
	return nil
}
