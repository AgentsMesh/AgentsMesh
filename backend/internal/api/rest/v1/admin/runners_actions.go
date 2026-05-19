package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *RunnerHandler) DisableRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	oldRunner, _ := h.adminService.GetRunner(c.Request.Context(), runnerID)

	r, err := h.adminService.DisableRunner(c.Request.Context(), runnerID)
	if err != nil {
		if errors.Is(err, adminservice.ErrRunnerNotFound) {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to disable runner")
		return
	}

	h.logAction(c, admin.AuditActionRunnerDisable, admin.TargetTypeRunner, runnerID, oldRunner, r)

	c.JSON(http.StatusOK, runnerResponse(r))
}

func (h *RunnerHandler) EnableRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	oldRunner, _ := h.adminService.GetRunner(c.Request.Context(), runnerID)

	r, err := h.adminService.EnableRunner(c.Request.Context(), runnerID)
	if err != nil {
		if errors.Is(err, adminservice.ErrRunnerNotFound) {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		apierr.InternalError(c, "Failed to enable runner")
		return
	}

	h.logAction(c, admin.AuditActionRunnerEnable, admin.TargetTypeRunner, runnerID, oldRunner, r)

	c.JSON(http.StatusOK, runnerResponse(r))
}

func (h *RunnerHandler) DeleteRunner(c *gin.Context) {
	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	deletedRunner, err := h.adminService.DeleteRunner(c.Request.Context(), runnerID)
	if err != nil {
		if errors.Is(err, adminservice.ErrRunnerNotFound) {
			apierr.ResourceNotFound(c, "Runner not found")
			return
		}
		if errors.Is(err, adminservice.ErrRunnerHasActivePods) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Cannot delete runner with active pods")
			return
		}
		apierr.InternalError(c, "Failed to delete runner")
		return
	}

	h.logAction(c, admin.AuditActionRunnerDelete, admin.TargetTypeRunner, runnerID, deletedRunner, nil)

	c.JSON(http.StatusOK, gin.H{"message": "Runner deleted successfully"})
}
