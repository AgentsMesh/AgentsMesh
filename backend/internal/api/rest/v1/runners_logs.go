package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequestLogUpload triggers a log upload on a runner.
// POST /api/v1/organizations/:slug/runners/:id/logs/upload
func (h *RunnerHandler) RequestLogUpload(c *gin.Context) {
	if h.logUploadSender == nil || h.logUploadService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Log upload service not configured")
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner exists and belongs to organization
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
	if !h.logUploadSender.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	// Create upload request (DB record + presigned URL)
	req, err := h.logUploadService.RequestUpload(c.Request.Context(), tenant.OrganizationID, runnerID, tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to create log upload request")
		slog.Error("Failed to create log upload request", "error", err)
		return
	}

	// Send upload command to runner; mark record as failed on error
	if err := h.logUploadSender.SendUploadLogs(runnerID, req.RequestID, req.PresignedURL, req.ExpiresAt); err != nil {
		h.logUploadService.MarkFailed(req.RequestID, "failed to send command to runner")

		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner disconnected before command could be sent")
		} else {
			apierr.InternalError(c, "Failed to send log upload command")
		}
		return
	}

	slog.Info("Log upload requested",
		"runner_id", runnerID,
		"request_id", req.RequestID,
		"user_id", tenant.UserID,
		"org_id", tenant.OrganizationID,
	)

	c.JSON(http.StatusAccepted, gin.H{
		"request_id": req.RequestID,
		"message":    "Log upload command sent to runner",
	})
}

// ListRunnerLogs returns diagnostic log records for a runner.
// GET /api/v1/organizations/:slug/runners/:id/logs
func (h *RunnerHandler) ListRunnerLogs(c *gin.Context) {
	if h.logUploadService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Log upload service not configured")
		return
	}

	runnerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid runner ID")
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify runner exists and belongs to organization
	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil || r.OrganizationID != tenant.OrganizationID {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	// Check visibility
	if r.Visibility == "private" && (r.RegisteredByUserID == nil || *r.RegisteredByUserID != tenant.UserID) {
		apierr.ForbiddenAccess(c)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, err := h.logUploadService.ListByRunner(c.Request.Context(), tenant.OrganizationID, runnerID, limit, offset)
	if err != nil {
		apierr.InternalError(c, "Failed to list runner logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}
