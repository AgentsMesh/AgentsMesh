package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpgradeRunnerRequest represents the request body for runner upgrade.
type UpgradeRunnerRequest struct {
	TargetVersion string `json:"target_version"`
	Force         bool   `json:"force"`
}

// UpgradeRunner triggers a remote upgrade on a runner.
// POST /api/v1/organizations/:slug/runners/:id/upgrade
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
		// Allow empty body (upgrade to latest)
		req = UpgradeRunnerRequest{}
	}

	tenant := middleware.GetTenant(c)

	// Verify runner exists and belongs to organization.
	// Return 404 for both "not found" and "wrong org" to prevent cross-org enumeration.
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil || r.OrganizationID != tenant.OrganizationID {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	// Check visibility: private runners are only visible to the registrant
	if r.Visibility == "private" && (r.RegisteredByUserID == nil || *r.RegisteredByUserID != tenant.UserID) {
		apierr.ForbiddenAccess(c)
		return
	}

	// Check if runner is online
	if !h.upgradeCommandSender.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	// Check pod count (unless force)
	if !req.Force && r.CurrentPods > 0 {
		apierr.Conflict(c, apierr.HAS_REFERENCES, "Runner has active pods. Use force=true to override.")
		return
	}

	// Generate request ID and send upgrade command
	requestID := uuid.New().String()
	if err := h.upgradeCommandSender.SendUpgradeRunner(runnerID, requestID, req.TargetVersion, req.Force); err != nil {
		// Differentiate error types for better client diagnostics
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner disconnected before command could be sent")
		} else {
			apierr.InternalError(c, "Failed to send upgrade command")
		}
		return
	}

	// Audit log
	slog.Info("Runner upgrade initiated",
		"runner_id", runnerID,
		"request_id", requestID,
		"target_version", req.TargetVersion,
		"force", req.Force,
		"user_id", tenant.UserID,
		"org_id", tenant.OrganizationID,
	)

	c.JSON(http.StatusAccepted, gin.H{
		"request_id": requestID,
		"message":    "Upgrade command sent to runner",
	})
}
