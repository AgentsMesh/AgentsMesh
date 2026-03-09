package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ChannelMarkReadRequest represents a mark-as-read request for a channel
type ChannelMarkReadRequest struct {
	MessageID int64 `json:"message_id" binding:"required"`
}

// MarkRead marks a channel as read up to a specific message
// POST /api/v1/organizations/:slug/channels/:id/read
func (h *ChannelHandler) MarkRead(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req ChannelMarkReadRequest
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

	if err := h.channelService.MarkRead(c.Request.Context(), channelID, tenant.UserID, req.MessageID); err != nil {
		apierr.InternalError(c, "Failed to mark channel as read")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetUnreadCounts returns unread message counts for all channels the user is a member of
// GET /api/v1/organizations/:slug/channels/unread
func (h *ChannelHandler) GetUnreadCounts(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	counts, err := h.channelService.GetUnreadCounts(c.Request.Context(), tenant.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to get unread counts")
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread": counts})
}

// MuteChannel mutes/unmutes a channel for the current user
// POST /api/v1/organizations/:slug/channels/:id/mute
func (h *ChannelHandler) MuteChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req struct {
		Muted bool `json:"muted"`
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

	if err := h.channelService.SetMemberMuted(c.Request.Context(), channelID, tenant.UserID, req.Muted); err != nil {
		apierr.InternalError(c, "Failed to update mute setting")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
