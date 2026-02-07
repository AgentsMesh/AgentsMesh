package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
)

// ListRunners lists runners in organization
// GET /api/v1/organizations/:slug/runners
func (h *RunnerHandler) ListRunners(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	runners, err := h.runnerService.ListRunners(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list runners"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runners": runners})
}

// GetRunner returns runner by ID
// GET /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) GetRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get relay connections if pod coordinator is available
	var relayConnections []runner.RelayConnectionInfo
	if h.podCoordinator != nil {
		relayConnections = h.podCoordinator.GetRelayConnections(runnerID)
	}

	c.JSON(http.StatusOK, gin.H{
		"runner":            r,
		"relay_connections": relayConnections,
	})
}

// UpdateRunner updates a runner
// PUT /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) UpdateRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	var req UpdateRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

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

	// Update runner
	updated, err := h.runnerService.UpdateRunner(c.Request.Context(), runnerID, runner.RunnerUpdateInput{
		Description:       req.Description,
		MaxConcurrentPods: req.MaxConcurrentPods,
		IsEnabled:         req.IsEnabled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runner": updated})
}

// DeleteRunner deletes a runner
// DELETE /api/v1/organizations/:slug/runners/:id
func (h *RunnerHandler) DeleteRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid runner ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin permission required"})
		return
	}

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Runner not found"})
		return
	}

	if r.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.runnerService.DeleteRunner(c.Request.Context(), runnerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete runner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted"})
}
