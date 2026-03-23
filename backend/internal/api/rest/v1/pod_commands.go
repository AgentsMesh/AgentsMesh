package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SendPodPrompt sends a prompt to a pod (mode-transparent).
// PTY: writes prompt text to stdin. ACP: sends prompt via ACP protocol.
// POST /api/v1/orgs/:slug/pods/:key/prompt
func (h *PodHandler) SendPodPrompt(c *gin.Context) {
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

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "pod is not active")
		return
	}

	var req struct {
		Prompt string `json:"prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "prompt is required")
		return
	}

	if h.commandSender == nil {
		apierr.InternalError(c, "Command sender not configured")
		return
	}

	if err := h.commandSender.SendPrompt(c.Request.Context(), pod.RunnerID, podKey, req.Prompt); err != nil {
		apierr.InternalError(c, "Failed to send prompt")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}
