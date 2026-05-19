package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

type updatePodAliasRequest struct {
	Alias *string `json:"alias"`
}

func (h *PodHandler) UpdatePodAlias(c *gin.Context) {
	podKey := c.Param("key")

	pod, err := h.podService.GetPod(c.Request.Context(), podKey)
	if err != nil {
		apierr.ResourceNotFound(c, "Pod not found")
		return
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)
	if !policy.PodPolicy.AllowWrite(sub, h.podResourceWithGrants(c.Request.Context(), podKey, pod.OrganizationID, pod.CreatedByID)) {
		apierr.ForbiddenAccess(c)
		return
	}

	var req updatePodAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if req.Alias != nil && strings.TrimSpace(*req.Alias) == "" {
		req.Alias = nil
	}

	if req.Alias != nil && len(*req.Alias) > 100 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Alias must be 100 characters or less")
		return
	}

	if err := h.podService.UpdateAlias(c.Request.Context(), podKey, req.Alias); err != nil {
		apierr.InternalError(c, "Failed to update pod alias")
		return
	}

	if h.eventBus != nil {
		aliasData, _ := json.Marshal(eventbus.PodAliasChangedData{
			PodKey: podKey,
			Alias:  req.Alias,
		})
		h.eventBus.Publish(c.Request.Context(), &eventbus.Event{
			Type:           eventbus.EventPodAliasChanged,
			Category:       eventbus.CategoryEntity,
			OrganizationID: tenant.OrganizationID,
			EntityType:     "pod",
			EntityID:       podKey,
			Data:           json.RawMessage(aliasData),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod alias updated"})
}
