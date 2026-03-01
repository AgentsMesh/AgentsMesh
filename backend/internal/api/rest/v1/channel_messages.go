package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
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

	messages, err := h.channelService.GetMessages(c.Request.Context(), channelID, nil, limit)
	if err != nil {
		apierr.InternalError(c, "Failed to list messages")
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessageRequest represents message send request
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
	PodKey  string `json:"pod_key"`
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

	msg, err := h.channelService.SendMessage(c.Request.Context(), channelID, podKey, &tenant.UserID, "text", req.Content, nil)
	if err != nil {
		apierr.InternalError(c, "Failed to send message")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": msg})
}
