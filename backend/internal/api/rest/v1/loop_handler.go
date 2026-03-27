package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// PodTerminatorForLoop defines the minimal interface needed by LoopHandler
// to terminate Pods (used for cancel run). Follows ISP — handler only needs TerminatePod.
type PodTerminatorForLoop interface {
	TerminatePod(ctx context.Context, podKey string) error
}

// LoopHandler handles loop-related requests
type LoopHandler struct {
	loopService    *loopService.LoopService
	loopRunService *loopService.LoopRunService
	orchestrator   *loopService.LoopOrchestrator
	podTerminator  PodTerminatorForLoop
}

// NewLoopHandler creates a new loop handler
func NewLoopHandler(
	ls *loopService.LoopService,
	lrs *loopService.LoopRunService,
	orch *loopService.LoopOrchestrator,
	podTerminator PodTerminatorForLoop,
) *LoopHandler {
	return &LoopHandler{
		loopService:    ls,
		loopRunService: lrs,
		orchestrator:   orch,
		podTerminator:  podTerminator,
	}
}

// ========== Loop CRUD ==========

// ListLoops lists loops for an organization
// GET /api/v1/orgs/:slug/loops
func (h *LoopHandler) ListLoops(c *gin.Context) {
	var req listLoopsQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	loops, total, err := h.loopService.List(c.Request.Context(), &loopService.ListLoopsFilter{
		OrganizationID: tenant.OrganizationID,
		Status:         req.Status,
		ExecutionMode:  req.ExecutionMode,
		CronEnabled:    req.CronEnabled,
		Query:          req.Query,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list loops")
		return
	}

	// Enrich with active run counts (H2)
	if len(loops) > 0 {
		loopIDs := make([]int64, len(loops))
		for i, l := range loops {
			loopIDs[i] = l.ID
		}
		if counts, err := h.loopRunService.CountActiveRunsByLoopIDs(c.Request.Context(), loopIDs); err == nil {
			for _, l := range loops {
				if count, ok := counts[l.ID]; ok {
					l.ActiveRunCount = int(count)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"loops":  loops,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

// GetLoop gets a loop by slug
// GET /api/v1/orgs/:slug/loops/:loop_slug
func (h *LoopHandler) GetLoop(c *gin.Context) {
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

	// Enrich with active run count (H2)
	if counts, err := h.loopRunService.CountActiveRunsByLoopIDs(c.Request.Context(), []int64{loop.ID}); err == nil {
		if count, ok := counts[loop.ID]; ok {
			loop.ActiveRunCount = int(count)
		}
	}

	// Enrich with average duration (M5)
	if avg, err := h.loopRunService.GetAvgDuration(c.Request.Context(), loop.ID); err == nil && avg != nil {
		loop.AvgDurationSec = avg
	}

	c.JSON(http.StatusOK, gin.H{"loop": loop})
}

// DeleteLoop deletes a loop.
// Atomically rejects deletion if there are active (pending/running) runs to prevent orphaned Pods.
// DELETE /api/v1/orgs/:slug/loops/:loop_slug
func (h *LoopHandler) DeleteLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	if err := h.loopService.Delete(c.Request.Context(), tenant.OrganizationID, loopSlug); err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else if errors.Is(err, loopService.ErrHasActiveRuns) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED,
				"Cannot delete loop with active runs. Cancel or wait for runs to complete first.")
		} else {
			apierr.InternalError(c, "Failed to delete loop")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Loop deleted"})
}
