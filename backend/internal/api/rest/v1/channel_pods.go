package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *ChannelHandler) JoinPod(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req struct {
		PodKey string `json:"pod_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if err := h.channelService.JoinChannel(c.Request.Context(), ch.ID, req.PodKey); err != nil {
		apierr.InternalError(c, "Failed to join channel")
		return
	}

	tenant := middleware.GetTenant(c)
	enriched, err := h.channelService.GetChannelForUser(c.Request.Context(), ch.ID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Pod joined channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pod joined channel", "channel": enriched})
}

func (h *ChannelHandler) LeavePod(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	podKey := c.Param("pod_key")
	if err := h.channelService.LeaveChannel(c.Request.Context(), ch.ID, podKey); err != nil {
		apierr.InternalError(c, "Failed to leave channel")
		return
	}

	tenant := middleware.GetTenant(c)
	enriched, err := h.channelService.GetChannelForUser(c.Request.Context(), ch.ID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Pod left channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pod left channel", "channel": enriched})
}

func (h *ChannelHandler) ListChannelPods(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	pods, err := h.channelService.GetChannelPods(c.Request.Context(), ch.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to list pods")
		return
	}

	c.JSON(http.StatusOK, gin.H{"pods": pods, "total": len(pods)})
}
