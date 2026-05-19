package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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

	if message.SenderPod != podKey && message.ReceiverPod != podKey {
		apierr.ForbiddenAccess(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

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
