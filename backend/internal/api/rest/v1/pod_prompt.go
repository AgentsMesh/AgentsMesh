package v1

import (
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// sendPromptRequest represents the request body for sending a prompt to a pod
type sendPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// SendPrompt sends a prompt to an active pod
// POST /api/v1/organizations/:slug/pods/:key/prompt
func (h *PodHandler) SendPrompt(c *gin.Context) {
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

	// Only creator or admin/owner can send prompts
	if pod.CreatedByID != tenant.UserID && tenant.UserRole == "member" {
		apierr.ForbiddenAdmin(c)
		return
	}

	if !pod.IsActive() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not active")
		return
	}

	var req sendPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		apierr.ValidationError(c, "Prompt must not be empty")
		return
	}

	if h.podCoordinator == nil {
		apierr.InternalError(c, "Pod coordinator not configured")
		return
	}

	commandSender := h.podCoordinator.GetCommandSender()
	if commandSender == nil {
		apierr.InternalError(c, "Command sender not configured")
		return
	}

	if err := commandSender.SendPrompt(c.Request.Context(), pod.RunnerID, podKey, req.Prompt); err != nil {
		apierr.InternalError(c, "Failed to send prompt to pod")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prompt sent"})
}
