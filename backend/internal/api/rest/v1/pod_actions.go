package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// TerminatePod terminates a pod
// POST /api/v1/organizations/:slug/pods/:key/terminate
func (h *PodHandler) TerminatePod(c *gin.Context) {
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

	// Only creator or admin can terminate
	if pod.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		apierr.ForbiddenAdmin(c)
		return
	}

	if err := h.podCoordinator.TerminatePod(c.Request.Context(), podKey); err != nil {
		if err == runner.ErrPodAlreadyTerminated {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod already terminated")
			return
		}
		apierr.InternalError(c, "Failed to terminate pod")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod terminated"})
}
