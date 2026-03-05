package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListRunners lists runners in organization
// GET /api/v1/organizations/:slug/runners
func (h *RunnerHandler) ListRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListRunners(c.Request.Context(), tenant.OrganizationID, tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to list runners")
		return
	}

	resp := gin.H{"runners": runners}
	if h.versionChecker != nil {
		if latestVersion := h.versionChecker.GetLatestVersion(c.Request.Context()); latestVersion != "" {
			resp["latest_runner_version"] = latestVersion
		}
	}
	c.JSON(http.StatusOK, resp)
}

// GetRunner returns runner by ID
// GET /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) GetRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if r.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check visibility: private runners are only visible to the registrant
	if r.Visibility == "private" && (r.RegisteredByUserID == nil || *r.RegisteredByUserID != tenant.UserID) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Get relay connections if pod coordinator is available
	var relayConnections []runner.RelayConnectionInfo
	if h.podCoordinator != nil {
		relayConnections = h.podCoordinator.GetRelayConnections(runnerID)
	}

	resp := gin.H{
		"runner":            r,
		"relay_connections": relayConnections,
	}
	if h.versionChecker != nil {
		if latestVersion := h.versionChecker.GetLatestVersion(c.Request.Context()); latestVersion != "" {
			resp["latest_runner_version"] = latestVersion
		}
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateRunner updates a runner
// PUT /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) UpdateRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	var req UpdateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

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

	// Update runner
	updated, err := h.runnerService.UpdateRunner(c.Request.Context(), runnerID, runner.RunnerUpdateInput{
		Description:       req.Description,
		MaxConcurrentPods: req.MaxConcurrentPods,
		IsEnabled:         req.IsEnabled,
		Visibility:        req.Visibility,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to update runner")
		return
	}

	c.JSON(http.StatusOK, gin.H{"runner": updated})
}

// DeleteRunner deletes a runner
// DELETE /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) DeleteRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if err := h.runnerService.DeleteRunner(c.Request.Context(), runnerID); err != nil {
		if errors.Is(err, runner.ErrRunnerHasLoopRefs) {
			apierr.Conflict(c, apierr.HAS_REFERENCES, "Cannot delete runner: it is referenced by one or more loops")
			return
		}
		apierr.InternalError(c, "Failed to delete runner")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted"})
}
