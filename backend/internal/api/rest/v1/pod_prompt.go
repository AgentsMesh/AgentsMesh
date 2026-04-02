package v1

import (
	"errors"
	"net/http"
	"strings"

	agentpodDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	orgDomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	runnersvc "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// maxPromptLength is the maximum allowed prompt size in bytes (64 KB).
const maxPromptLength = 64 * 1024

// sendPromptRequest represents the request body for sending a prompt to a pod.
type sendPromptRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// SendPrompt sends a prompt to an active pod via terminal input.
// POST /api/v1/orgs/:slug/pods/:key/prompt
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

	// Only creator or admin/owner can send prompts.
	if pod.CreatedByID != tenant.UserID &&
		tenant.UserRole != orgDomain.RoleOwner && tenant.UserRole != orgDomain.RoleAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	if pod.Status != agentpodDomain.StatusRunning {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Pod is not running")
		return
	}

	var req sendPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		apierr.ValidationError(c, "Prompt must not be empty")
		return
	}

	if len(req.Prompt) > maxPromptLength {
		apierr.ValidationError(c, "Prompt exceeds maximum length")
		return
	}

	if h.terminalRouter == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal input service is not available")
		return
	}

	// Send prompt text and Enter (\r) as two separate writes.
	// Some TUIs (e.g. Codex) treat a single combined write as a paste operation and
	// do not submit the input; splitting into two writes ensures the Enter is recognized
	// as a discrete keypress that triggers submission.
	if sendTerminalInput(c, h.terminalRouter, podKey, []byte(req.Prompt), "Failed to send prompt to pod") {
		return
	}
	if sendTerminalInput(c, h.terminalRouter, podKey, []byte("\r"), "Failed to submit prompt to pod") {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prompt sent"})
}

// sendTerminalInput sends data via the terminal router and writes an error response on failure.
// Returns true if an error was handled (caller should return), false on success.
func sendTerminalInput(c *gin.Context, router PodTerminalRouter, podKey string, data []byte, failMsg string) bool {
	err := router.RouteInput(podKey, data)
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, runnersvc.ErrRunnerNotConnected):
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner for pod is not connected")
	case errors.Is(err, runnersvc.ErrCommandSenderNotSet):
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Terminal input service is not available")
	default:
		apierr.InternalError(c, failMsg)
	}
	return true
}
