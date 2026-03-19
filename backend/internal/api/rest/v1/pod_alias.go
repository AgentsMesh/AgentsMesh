package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// updatePodAliasRequest represents the request body for updating a pod alias
type updatePodAliasRequest struct {
	// Alias is a pointer to distinguish between null (clear) and absent (no change).
	// JSON: {"alias": "my-name"} sets the alias; {"alias": null} clears it.
	Alias *string `json:"alias"`
}

// UpdatePodAlias updates the alias for a pod
// PATCH /api/v1/organizations/:slug/pods/:key/alias
func (h *PodHandler) UpdatePodAlias(c *gin.Context) {
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

	// Only creator or admin/owner can update alias
	if pod.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		apierr.ForbiddenAdmin(c)
		return
	}

	var req updatePodAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Treat empty string as null (clear alias)
	if req.Alias != nil && strings.TrimSpace(*req.Alias) == "" {
		req.Alias = nil
	}

	// Validate alias length when not clearing
	if req.Alias != nil && len(*req.Alias) > 100 {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Alias must be 100 characters or less")
		return
	}

	if err := h.podService.UpdateAlias(c.Request.Context(), podKey, req.Alias); err != nil {
		apierr.InternalError(c, "Failed to update pod alias")
		return
	}

	// Publish pod:alias_changed event for real-time sync
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
