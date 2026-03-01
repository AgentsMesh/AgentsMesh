package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// GetMessages handles GET /messages
// @Summary Get messages for the current pod
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param unread_only query bool false "Only return unread messages"
// @Param message_types query []string false "Filter by message types"
// @Param limit query int false "Maximum messages to return" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /messages [get]
func (h *MessageHandler) GetMessages(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	unreadOnly := c.Query("unread_only") == "true"
	messageTypes := c.QueryArray("message_types")

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := h.msgSvc.GetMessages(c.Request.Context(), podKey, unreadOnly, messageTypes, limit, offset)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	unreadCount, _ := h.msgSvc.GetUnreadCount(c.Request.Context(), podKey)

	c.JSON(http.StatusOK, gin.H{
		"messages":     messages,
		"total":        len(messages),
		"unread_count": unreadCount,
	})
}

// GetUnreadCount handles GET /messages/unread-count
// @Summary Get count of unread messages
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Success 200 {object} map[string]interface{}
// @Router /messages/unread-count [get]
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	count, err := h.msgSvc.GetUnreadCount(c.Request.Context(), podKey)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetMessage handles GET /messages/:id
// @Summary Get a specific message by ID
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param id path int true "Message ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /messages/{id} [get]
func (h *MessageHandler) GetMessage(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "invalid message ID")
		return
	}

	message, err := h.msgSvc.GetMessage(c.Request.Context(), messageID)
	if err != nil {
		apierr.ResourceNotFound(c, "message not found")
		return
	}

	// Check access
	if message.SenderPod != podKey && message.ReceiverPod != podKey {
		apierr.ForbiddenAccess(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// GetConversation handles GET /messages/conversation/:correlation_id
// @Summary Get all messages in a conversation by correlation ID
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param correlation_id path string true "Correlation ID"
// @Param limit query int false "Maximum messages to return" default(100)
// @Success 200 {object} map[string]interface{}
// @Router /messages/conversation/{correlation_id} [get]
func (h *MessageHandler) GetConversation(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	correlationID := c.Param("correlation_id")
	if correlationID == "" {
		apierr.BadRequest(c, apierr.MISSING_REQUIRED, "correlation_id is required")
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	messages, err := h.msgSvc.GetConversation(c.Request.Context(), correlationID, limit)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	// Filter to messages involving this pod
	var filtered []*agent.AgentMessage
	for _, m := range messages {
		if m.SenderPod == podKey || m.ReceiverPod == podKey {
			filtered = append(filtered, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": filtered,
		"total":    len(filtered),
	})
}

// GetSentMessages handles GET /messages/sent
// @Summary Get messages sent by this pod
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param limit query int false "Maximum messages to return" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /messages/sent [get]
func (h *MessageHandler) GetSentMessages(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := h.msgSvc.GetSentMessages(c.Request.Context(), podKey, limit, offset)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    len(messages),
	})
}
