package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// JoinPod joins a pod to a channel
// POST /api/v1/organizations/:slug/channels/:id/pods
func (h *ChannelHandler) JoinPod(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req struct {
		PodKey string `json:"pod_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		apierr.ResourceNotFound(c, "Channel not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if err := h.channelService.JoinChannel(c.Request.Context(), channelID, req.PodKey); err != nil {
		apierr.InternalError(c, "Failed to join channel")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod joined channel"})
}

// LeavePod removes a pod from a channel
// DELETE /api/v1/organizations/:slug/channels/:id/pods/:pod_key
func (h *ChannelHandler) LeavePod(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	podKey := c.Param("pod_key")

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		apierr.ResourceNotFound(c, "Channel not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	if err := h.channelService.LeaveChannel(c.Request.Context(), channelID, podKey); err != nil {
		apierr.InternalError(c, "Failed to leave channel")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod left channel"})
}

// ListChannelPods returns pods joined to a channel
// GET /api/v1/organizations/:slug/channels/:id/pods
func (h *ChannelHandler) ListChannelPods(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		apierr.ResourceNotFound(c, "Channel not found")
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return
	}

	pods, err := h.channelService.GetChannelPods(c.Request.Context(), channelID)
	if err != nil {
		apierr.InternalError(c, "Failed to list pods")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pods":  pods,
		"total": len(pods),
	})
}
