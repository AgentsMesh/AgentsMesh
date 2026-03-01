package admin

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// DisableRunner disables a runner
func (h *RunnerHandler) DisableRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	// Get old data for audit log
	oldRunner, _ := h.adminService.GetRunner(c.Request.Context(), runnerID)

	r, err := h.adminService.DisableRunner(c.Request.Context(), runnerID)
	if err != nil {
		if err == adminservice.ErrRunnerNotFound {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to disable runner")
		return
	}

	// Log disable action
	h.logAction(c, admin.AuditActionRunnerDisable, admin.TargetTypeRunner, runnerID, oldRunner, r)

	c.JSON(http.StatusOK, runnerResponse(r))
}

// EnableRunner enables a runner
func (h *RunnerHandler) EnableRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	// Get old data for audit log
	oldRunner, _ := h.adminService.GetRunner(c.Request.Context(), runnerID)

	r, err := h.adminService.EnableRunner(c.Request.Context(), runnerID)
	if err != nil {
		if err == adminservice.ErrRunnerNotFound {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to enable runner")
		return
	}

	// Log enable action
	h.logAction(c, admin.AuditActionRunnerEnable, admin.TargetTypeRunner, runnerID, oldRunner, r)

	c.JSON(http.StatusOK, runnerResponse(r))
}

// DeleteRunner deletes a runner
func (h *RunnerHandler) DeleteRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	deletedRunner, err := h.adminService.DeleteRunner(c.Request.Context(), runnerID)
	if err != nil {
		if err == adminservice.ErrRunnerNotFound {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		if err == adminservice.ErrRunnerHasActivePods {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Cannot delete runner with active pods")
			return
		}
		apierr.InternalError(c, "Failed to delete runner")
		return
	}

	// Log delete action
	h.logAction(c, admin.AuditActionRunnerDelete, admin.TargetTypeRunner, runnerID, deletedRunner, nil)

	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted successfully"})
}
