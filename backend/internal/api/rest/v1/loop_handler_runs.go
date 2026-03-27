package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListRuns lists runs for a loop.
// GET /api/v1/orgs/:slug/loops/:loop_slug/runs
func (h *LoopHandler) ListRuns(c *gin.Context) {
	var req listRunsQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	runsOffset := req.Offset
	if runsOffset < 0 {
		runsOffset = 0
	}

	runs, total, err := h.loopRunService.ListRuns(c.Request.Context(), &loopService.ListRunsFilter{
		LoopID: loop.ID,
		Status: req.Status,
		Limit:  limit,
		Offset: runsOffset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list runs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":   runs,
		"total":  total,
		"limit":  limit,
		"offset": runsOffset,
	})
}

// GetRun gets a run by ID.
// GET /api/v1/orgs/:slug/loops/:loop_slug/runs/:run_id
func (h *LoopHandler) GetRun(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")
	runIDStr := c.Param("run_id")

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		apierr.ValidationError(c, "Invalid run ID")
		return
	}

	run, err := h.loopRunService.GetByID(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, loopService.ErrRunNotFound) {
			apierr.ResourceNotFound(c, "Run not found")
		} else {
			apierr.InternalError(c, "Failed to get run")
		}
		return
	}

	if run.LoopID != loop.ID {
		apierr.ResourceNotFound(c, "Run not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"run": run})
}

// CancelRun cancels a running loop run.
// POST /api/v1/orgs/:slug/loops/:loop_slug/runs/:run_id/cancel
func (h *LoopHandler) CancelRun(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")
	runIDStr := c.Param("run_id")

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		apierr.ValidationError(c, "Invalid run ID")
		return
	}

	run, err := h.loopRunService.GetByID(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, loopService.ErrRunNotFound) {
			apierr.ResourceNotFound(c, "Run not found")
		} else {
			apierr.InternalError(c, "Failed to get run")
		}
		return
	}

	if run.LoopID != loop.ID {
		apierr.ResourceNotFound(c, "Run not found")
		return
	}

	if run.IsTerminal() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Run is already in terminal state")
		return
	}

	// SSOT: cancel by terminating the Pod — run status will be derived from Pod state
	if run.PodKey != nil && h.podTerminator != nil {
		if err := h.podTerminator.TerminatePod(c.Request.Context(), *run.PodKey); err != nil {
			apierr.InternalError(c, "Failed to terminate pod")
			return
		}
	} else {
		if err := h.orchestrator.MarkRunCancelled(c.Request.Context(), runID, "Cancelled by user"); err != nil {
			apierr.InternalError(c, "Failed to cancel run")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Run cancelled"})
}
