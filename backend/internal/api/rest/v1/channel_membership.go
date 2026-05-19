package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *ChannelHandler) JoinChannel(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.JoinPublicChannel(c.Request.Context(), ch.ID, tenant.UserID); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "joined"})
}

func (h *ChannelHandler) LeaveChannel(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.LeaveUserChannel(c.Request.Context(), ch.ID, tenant.UserID); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "left"})
}

type inviteMembersRequest struct {
	UserIDs []int64 `json:"user_ids" binding:"required"`
}

// InviteMembers adds users to a channel. The caller must be an existing member.
// POST /api/v1/organizations/:slug/channels/:id/members
func (h *ChannelHandler) InviteMembers(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req inviteMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.InviteMembers(c.Request.Context(), ch.ID, tenant.UserID, req.UserIDs); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "invited"})
}

func (h *ChannelHandler) RemoveMember(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	targetUserID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	tenant := middleware.GetTenant(c)
	if err := h.channelService.RemoveMember(c.Request.Context(), ch.ID, tenant.UserID, targetUserID); err != nil {
		handleChannelServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}
