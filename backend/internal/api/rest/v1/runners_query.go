package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListAvailableRunners lists available runners for pods
// GET /api/v1/organizations/:slug/runners/available
func (h *RunnerHandler) ListAvailableRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListAvailableRunners(c.Request.Context(), tenant.OrganizationID, tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to list runners")
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

// ListRunnerPods lists pods for a specific runner
// GET /api/v1/organizations/:slug/runners/:id/pods
func (h *RunnerHandler) ListRunnerPods(c *gin.Context) {
	if h.podService == nil {
		apierr.InternalError(c, "Pod service not configured")
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	var req ListRunnerPodsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check visibility: private runners are only visible to the registrant
	if r.Visibility == "private" && (r.RegisteredByUserID == nil || *r.RegisteredByUserID != tenant.UserID) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Default limit
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	pods, total, err := h.podService.ListPodsByRunner(c.Request.Context(), runnerID, req.Status, limit, req.Offset)
	if err != nil {
		apierr.InternalError(c, "Failed to list pods")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pods":   pods,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

// QuerySandboxes queries sandbox status for specified pod keys on a runner
// POST /api/v1/organizations/:slug/runners/:id/sandboxes/query
func (h *RunnerHandler) QuerySandboxes(c *gin.Context) {
	if h.sandboxQueryService == nil || h.sandboxQuerySender == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Sandbox query service not configured")
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	var req QuerySandboxesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check visibility: private runners are only visible to the registrant
	if r.Visibility == "private" && (r.RegisteredByUserID == nil || *r.RegisteredByUserID != tenant.UserID) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check if runner is connected
	if !h.sandboxQuerySender.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	// Query sandboxes
	result, err := h.sandboxQueryService.QuerySandboxes(
		c.Request.Context(),
		runnerID,
		req.PodKeys,
		h.sandboxQuerySender.SendQuerySandboxes,
	)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	if result.Error != "" {
		c.JSON(http.StatusOK, gin.H{
			"error":     result.Error,
			"sandboxes": result.Sandboxes,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sandboxes": result.Sandboxes,
	})
}
