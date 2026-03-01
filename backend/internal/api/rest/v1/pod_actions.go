package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
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

	if err := h.podService.TerminatePod(c.Request.Context(), podKey); err != nil {
		if err == ErrPodTerminated {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod already terminated")
			return
		}
		apierr.InternalError(c, "Failed to terminate pod")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod terminated"})
}

// SendPromptRequest represents prompt sending request
type SendPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// SendPrompt sends a prompt to the pod
// POST /api/v1/organizations/:slug/pods/:key/send-prompt
func (h *PodHandler) SendPrompt(c *gin.Context) {
	podKey := c.Param("key")

	var req SendPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

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

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	// TODO: Implement prompt sending via gRPC to runner
	// For now, return not implemented
	_ = req.Prompt
	apierr.NotImplemented(c, "Prompt sending via REST not implemented. Use terminal WebSocket.")
}
