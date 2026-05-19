package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type AutopilotControlRequest struct {
	Action               string `json:"action" binding:"required,oneof=pause resume stop approve takeover handback"`
	ContinueExecution    *bool  `json:"continue_execution,omitempty"`    // For approve action
	AdditionalIterations int32  `json:"additional_iterations,omitempty"` // For approve action
}

func (h *AutopilotControllerHandler) sendAutopilotControl(c *gin.Context, autopilotPod *agentpod.AutopilotController, action string, req *AutopilotControlRequest) {
	if h.commandSender == nil {
		apierr.InternalError(c, "command sender not configured")
		return
	}

	cmd := &runnerv1.AutopilotControlCommand{
		AutopilotKey: autopilotPod.AutopilotControllerKey,
	}

	switch action {
	case "pause":
		cmd.Action = &runnerv1.AutopilotControlCommand_Pause{Pause: &runnerv1.AutopilotPauseAction{}}
	case "resume":
		cmd.Action = &runnerv1.AutopilotControlCommand_Resume{Resume: &runnerv1.AutopilotResumeAction{}}
	case "stop":
		cmd.Action = &runnerv1.AutopilotControlCommand_Stop{Stop: &runnerv1.AutopilotStopAction{}}
	case "approve":
		continueExec := true
		if req != nil && req.ContinueExecution != nil {
			continueExec = *req.ContinueExecution
		}
		additionalIter := int32(0)
		if req != nil {
			additionalIter = req.AdditionalIterations
		}
		cmd.Action = &runnerv1.AutopilotControlCommand_Approve{
			Approve: &runnerv1.AutopilotApproveAction{
				ContinueExecution:    continueExec,
				AdditionalIterations: additionalIter,
			},
		}
	case "takeover":
		cmd.Action = &runnerv1.AutopilotControlCommand_Takeover{Takeover: &runnerv1.AutopilotTakeoverAction{}}
	case "handback":
		cmd.Action = &runnerv1.AutopilotControlCommand_Handback{Handback: &runnerv1.AutopilotHandbackAction{}}
	default:
		apierr.InvalidInput(c, "invalid action")
		return
	}

	if err := h.commandSender.SendAutopilotControl(autopilotPod.RunnerID, cmd); err != nil {
		apierr.InternalError(c, "failed to send control command")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "action": action})
}

func (h *AutopilotControllerHandler) PauseAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}
	h.sendAutopilotControl(c, autopilotPod, "pause", nil)
}

func (h *AutopilotControllerHandler) ResumeAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}
	h.sendAutopilotControl(c, autopilotPod, "resume", nil)
}

func (h *AutopilotControllerHandler) StopAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}
	h.sendAutopilotControl(c, autopilotPod, "stop", nil)
}

func (h *AutopilotControllerHandler) ApproveAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}

	var req AutopilotControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Action = "approve"
	}

	h.sendAutopilotControl(c, autopilotPod, "approve", &req)
}

func (h *AutopilotControllerHandler) TakeoverAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}
	h.sendAutopilotControl(c, autopilotPod, "takeover", nil)
}

func (h *AutopilotControllerHandler) HandbackAutopilotController(c *gin.Context) {
	autopilotPod := h.getAutopilotControllerFromContext(c)
	if autopilotPod == nil {
		return
	}
	h.sendAutopilotControl(c, autopilotPod, "handback", nil)
}
