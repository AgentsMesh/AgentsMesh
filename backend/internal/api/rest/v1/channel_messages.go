package v1

import (
	"errors"
	"net/http"
	"strconv"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListMessages lists channel messages
// GET /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) ListMessages(c *gin.Context) {
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

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Cursor-based pagination: before_id takes precedence over time-based before
	var messages []*channelDomain.Message
	if beforeIDStr := c.Query("before_id"); beforeIDStr != "" {
		beforeID, err := strconv.ParseInt(beforeIDStr, 10, 64)
		if err != nil {
			apierr.InvalidInput(c, "Invalid before_id")
			return
		}
		messages, err = h.channelService.GetMessagesByCursor(c.Request.Context(), channelID, beforeID, limit)
		if err != nil {
			apierr.InternalError(c, "Failed to list messages")
			return
		}
	} else {
		var fetchErr error
		messages, fetchErr = h.channelService.GetMessages(c.Request.Context(), channelID, nil, limit)
		if fetchErr != nil {
			apierr.InternalError(c, "Failed to list messages")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessageRequest represents message send request
type SendMessageRequest struct {
	Content  string                        `json:"content" binding:"required"`
	PodKey   string                        `json:"pod_key"`
	Mentions []channelService.MentionInput `json:"mentions"`
}

// SendMessage sends a message to a channel
// POST /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) SendMessage(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req SendMessageRequest
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

	if ch.IsArchived {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot send messages to archived channel")
		return
	}

	// Determine sender pod key from request body (for user-initiated messages)
	var podKey *string
	if req.PodKey != "" {
		podKey = &req.PodKey
	}

	msg, err := h.channelService.SendMessage(c.Request.Context(), channelID, podKey, &tenant.UserID, "text", req.Content, nil, req.Mentions)
	if err != nil {
		apierr.InternalError(c, "Failed to send message")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": msg})
}

// EditMessageRequest represents message edit request
type EditMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// EditMessage edits a channel message
// PUT /api/v1/organizations/:slug/channels/:id/messages/:msg_id
func (h *ChannelHandler) EditMessage(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}
	msgID, err := strconv.ParseInt(c.Param("msg_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid message ID")
		return
	}

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	msg, err := h.channelService.EditMessage(c.Request.Context(), channelID, msgID, tenant.UserID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, channelService.ErrMessageNotFound):
			apierr.ResourceNotFound(c, "Message not found")
		case errors.Is(err, channelService.ErrNotMessageSender):
			apierr.ForbiddenAccess(c)
		case errors.Is(err, channelService.ErrChannelArchived):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot edit messages in archived channel")
		default:
			apierr.InternalError(c, "Failed to edit message")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// DeleteMessage soft-deletes a channel message
// DELETE /api/v1/organizations/:slug/channels/:id/messages/:msg_id
func (h *ChannelHandler) DeleteMessage(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}
	msgID, err := strconv.ParseInt(c.Param("msg_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid message ID")
		return
	}

	tenant := middleware.GetTenant(c)
	err = h.channelService.DeleteMessage(c.Request.Context(), channelID, msgID, tenant.UserID)
	if err != nil {
		switch {
		case errors.Is(err, channelService.ErrMessageNotFound):
			apierr.ResourceNotFound(c, "Message not found")
		case errors.Is(err, channelService.ErrNotMessageSender):
			apierr.ForbiddenAccess(c)
		case errors.Is(err, channelService.ErrChannelArchived):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot delete messages in archived channel")
		default:
			apierr.InternalError(c, "Failed to delete message")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
