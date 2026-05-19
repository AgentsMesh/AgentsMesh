package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpgradeRunnerRequest struct {
	TargetVersion string `json:"target_version"`
	// Deprecated: accepted for compat but ignored — Poddaemon keeps pods alive across Runner restarts.
	Force bool `json:"force,omitempty"`
}

func (h *RunnerHandler) UpgradeRunner(c *gin.Context) {
	if h.upgradeCommandSender == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Upgrade service not configured")
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	var req UpgradeRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = UpgradeRunnerRequest{}
	}

	tenant := middleware.GetTenant(c)
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if !policy.RunnerPolicy.AllowWrite(sub, policy.VisibleResource(
		r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	if req.Force {
		slog.WarnContext(c.Request.Context(), "Deprecated 'force' field received — ignored since Poddaemon upgrade path",
			"runner_id", runnerID, "user_id", tenant.UserID)
	}

	if !h.upgradeCommandSender.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	requestID := uuid.New().String()
	if err := h.upgradeCommandSender.SendUpgradeRunner(runnerID, requestID, req.TargetVersion, true); err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner disconnected before command could be sent")
		} else {
			apierr.InternalError(c, "Failed to send upgrade command")
		}
		return
	}

	slog.InfoContext(c.Request.Context(), "Runner upgrade initiated",
		"runner_id", runnerID,
		"request_id", requestID,
		"target_version", req.TargetVersion,
		"active_pod_count", r.CurrentPods,
		"user_id", tenant.UserID,
		"org_id", tenant.OrganizationID,
	)

	c.JSON(http.StatusAccepted, gin.H{
		"request_id": requestID,
		"message":    "Upgrade command sent to runner",
	})
}
