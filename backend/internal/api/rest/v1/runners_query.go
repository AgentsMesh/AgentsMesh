package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
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

	// Verify runner belongs to organization and is visible to the requester
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	subject := policy.From(tenant)
	ownerID := int64(0)
	if r.RegisteredByUserID != nil {
		ownerID = *r.RegisteredByUserID
	}
	if !policy.RunnerPolicy.AllowRead(subject, policy.ResourceContext{
		OrgID: r.OrganizationID, OwnerID: ownerID, Visibility: r.Visibility,
	}) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Default limit
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	// Members only see their own pods on this runner
	podOwnerFilter := policy.PodPolicy.FilterList(subject)
	pods, total, err := h.podService.ListPodsByRunner(c.Request.Context(), runnerID, req.Status, podOwnerFilter, limit, req.Offset)
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
	if h.sandboxQueryService == nil {
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

	// Verify runner belongs to organization and is visible to the requester
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	ownerID := int64(0)
	if r.RegisteredByUserID != nil {
		ownerID = *r.RegisteredByUserID
	}
	if !policy.RunnerPolicy.AllowRead(policy.From(tenant), policy.ResourceContext{
		OrgID: r.OrganizationID, OwnerID: ownerID, Visibility: r.Visibility,
	}) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check if runner is connected
	if !h.sandboxQueryService.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	// Query sandboxes
	result, err := h.sandboxQueryService.QuerySandboxes(
		c.Request.Context(),
		runnerID,
		req.PodKeys,
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
