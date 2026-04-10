package v1

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type updatePodPerpetualRequest struct {
	Perpetual bool `json:"perpetual"`
}

// UpdatePodPerpetual toggles perpetual mode for a pod.
// PATCH /api/v1/organizations/:slug/pods/:key/perpetual
func (h *PodHandler) UpdatePodPerpetual(c *gin.Context) {
	podKey := c.Param("key")

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if pod.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if pod.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		apierr.ForbiddenAdmin(c)
		return
	}

	var req updatePodPerpetualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.podService.UpdatePerpetual(c.Request.Context(), podKey, req.Perpetual); err != nil {
		apierr.InternalError(c, "Failed to update pod perpetual mode")
		return
	}

	// Notify runner so in-memory pod state is updated immediately.
	if h.commandSender != nil {
		h.notifyRunnerPerpetual(c.Request.Context(), pod.RunnerID, podKey, req.Perpetual)
	}

	if h.eventBus != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"pod_key":   podKey,
			"perpetual": req.Perpetual,
		})
		h.eventBus.Publish(c.Request.Context(), &eventbus.Event{
			Type:           eventbus.EventPodPerpetualChanged,
			Category:       eventbus.CategoryEntity,
			OrganizationID: tenant.OrganizationID,
			EntityType:     "pod",
			EntityID:       podKey,
			Data:           json.RawMessage(data),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod perpetual mode updated"})
}

func (h *PodHandler) notifyRunnerPerpetual(ctx context.Context, runnerID int64, podKey string, perpetual bool) {
	if err := h.commandSender.SendUpdatePodPerpetual(ctx, runnerID, podKey, perpetual); err != nil {
		// Non-fatal: DB is already updated; runner will use correct state on next restart.
		_ = err
	}
}
