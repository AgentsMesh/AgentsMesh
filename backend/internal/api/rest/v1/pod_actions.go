package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
)

func (h *PodHandler) TerminatePod(c *gin.Context) {
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

	if err := h.podCoordinator.TerminatePod(c.Request.Context(), podKey); err != nil {
		if errors.Is(err, runner.ErrPodAlreadyTerminated) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod already terminated")
			return
		}
		apierr.InternalError(c, "Failed to terminate pod")
		return
	}

	if h.grantService != nil {
		_ = h.grantService.CleanupByResource(c.Request.Context(), grant.TypePod, podKey)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod terminated"})
}
