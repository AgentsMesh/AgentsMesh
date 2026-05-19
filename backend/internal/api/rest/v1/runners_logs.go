package v1

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/anthropics/agentsmesh/backend/pkg/policy"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if !policy.RunnerPolicy.AllowRead(sub, h.runnerResourceWithGrants(
		c.Request.Context(), runnerID, r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
		apierr.ForbiddenAccess(c)
		return
	}

	if !h.logUploadSender.IsConnected(runnerID) {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner is not connected")
		return
	}

	req, err := h.logUploadService.RequestUpload(c.Request.Context(), tenant.OrganizationID, runnerID, tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to create log upload request")
		slog.ErrorContext(c.Request.Context(), "Failed to create log upload request", "error", err)
		return
	}

	if err := h.logUploadSender.SendUploadLogs(runnerID, req.RequestID, req.PresignedURL, req.ExpiresAt); err != nil {
		h.logUploadService.MarkFailed(req.RequestID, "failed to send command to runner")

		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "Runner disconnected before command could be sent")
		} else {
			apierr.InternalError(c, "Failed to send log upload command")
		}
		return
	}

	slog.InfoContext(c.Request.Context(), "Log upload requested",
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
	sub := policy.NewSubject(tenant.OrganizationID, tenant.UserID, tenant.UserRole)

	r, err := h.runnerService.GetRunner(c.Request.Context(), runnerID)
	if err != nil {
		apierr.ResourceNotFound(c, "Runner not found")
		return
	}

	if !policy.RunnerPolicy.AllowRead(sub, h.runnerResourceWithGrants(
		c.Request.Context(), runnerID, r.OrganizationID, r.RegisteredByUserID, r.Visibility,
	)) {
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
