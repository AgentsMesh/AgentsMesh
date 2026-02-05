package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ListAvailableRunners lists available runners for pods
// GET /api/v1/organizations/:slug/runners/available
func (h *RunnerHandler) ListAvailableRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListAvailableRunners(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runners"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

// ListRunnerPods lists pods for a specific runner
// GET /api/v1/organizations/:slug/runners/:id/pods
func (h *RunnerHandler) ListRunnerPods(c *gin.Context) {
	if h.podService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pod service not configured"})
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	var req ListRunnerPodsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Default limit
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	pods, total, err := h.podService.ListPodsByRunner(c.Request.Context(), runnerID, req.Status, limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pods"})
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
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Sandbox query service not configured"})
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	var req QuerySandboxesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if runner is connected
	if !h.sandboxQuerySender.IsConnected(runnerID) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Runner is not connected"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
