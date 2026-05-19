package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

func statusSlice(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func (h *RunnerHandler) ListAvailableRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	filter := policy.RunnerPolicy.ListFilter(sub)
	runners, err := h.runnerService.ListAvailableRunners(c.Request.Context(), tenant.OrganizationID, filter.VisibilityUserID)
	if err != nil {
		apierr.InternalError(c, "Failed to list runners")
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

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
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if !policy.RunnerPolicy.AllowRead(sub, h.runnerResourceWithGrants(
		c.Request.Context(), runnerID, r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	podFilter := policy.PodPolicy.ListFilter(sub)
	pods, total, err := h.podService.ListPodsByRunner(c.Request.Context(), runnerID, agentpod.PodListQuery{
		Statuses:      statusSlice(req.Status),
		CreatedByID:   podFilter.OwnerOnly,
		GrantedUserID: podFilter.GrantUserID,
		Limit:         limit,
		Offset:        req.Offset,
	})
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
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if !policy.RunnerPolicy.AllowRead(sub, h.runnerResourceWithGrants(
		c.Request.Context(), runnerID, r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	if !h.sandboxQueryService.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

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
