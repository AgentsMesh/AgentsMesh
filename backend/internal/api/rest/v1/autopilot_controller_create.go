package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
)

// CreateAutopilotController handles POST /autopilot-controllers
func (h *AutopilotControllerHandler) CreateAutopilotController(c *gin.Context) {
	orgID := getOrgID(c)
	if orgID == 0 {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "organization context required")
		return
	}

	var req CreateAutopilotControllerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Verify target pod exists
	if h.podService == nil {
		apierr.InternalError(c, "pod service not configured")
		return
	}

	targetPod, err := h.podService.GetPod(c.Request.Context(), req.PodKey)
	if err != nil {
		apierr.ResourceNotFound(c, "target pod not found")
		return
	}

	// Verify pod belongs to this organization
	if targetPod.OrganizationID != orgID {
		apierr.ForbiddenAccess(c)
		return
	}

	// Delegate to service for record creation + gRPC command
	if h.service == nil {
		apierr.InternalError(c, "autopilot service not configured")
		return
	}

	controller, err := h.service.CreateAndStart(c.Request.Context(), &agentpodSvc.CreateAndStartRequest{
		OrganizationID:        orgID,
		Pod:                   targetPod,
		InitialPrompt:         req.InitialPrompt,
		MaxIterations:         req.MaxIterations,
		IterationTimeoutSec:   req.IterationTimeoutSec,
		NoProgressThreshold:   req.NoProgressThreshold,
		SameErrorThreshold:    req.SameErrorThreshold,
		ApprovalTimeoutMin:    req.ApprovalTimeoutMin,
		ControlAgentSlug:      req.ControlAgentSlug,
		ControlPromptTemplate: req.ControlPromptTemplate,
		MCPConfigJSON:         req.MCPConfigJSON,
		KeyPrefix:             "autopilot",
	})
	if err != nil {
		apierr.InternalError(c, "failed to create autopilot controller")
		return
	}

	c.JSON(http.StatusCreated, toAutopilotControllerResponse(controller))
}
