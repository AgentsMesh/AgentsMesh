package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type ChannelMarkReadRequest struct {
	MessageID int64 `json:"message_id" binding:"required"`
}

func (h *ChannelHandler) MarkRead(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req ChannelMarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.MarkRead(c.Request.Context(), ch.ID, tenant.UserID, req.MessageID); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ChannelHandler) GetUnreadCounts(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	counts, err := h.channelService.GetUnreadCounts(c.Request.Context(), tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to get unread counts")
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread": counts})
}

func (h *ChannelHandler) MuteChannel(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req struct {
		Muted bool `json:"muted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.SetMemberMuted(c.Request.Context(), ch.ID, tenant.UserID, req.Muted); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
