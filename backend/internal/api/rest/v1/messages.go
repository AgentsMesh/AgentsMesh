package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	agentSvc "github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/gin-gonic/gin"
)

// MessageHandler handles agent message API endpoints
type MessageHandler struct {
	msgSvc *agentSvc.MessageService
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(msgSvc *agentSvc.MessageService) *MessageHandler {
	return &MessageHandler{
		msgSvc: msgSvc,
	}
}

// AgentSendMessageRequest represents a request to send an agent message
type AgentSendMessageRequest struct {
	ReceiverPod   string                 `json:"receiver_pod" binding:"required"`
	MessageType   string                 `json:"message_type" binding:"required"`
	Content       map[string]interface{} `json:"content" binding:"required"`
	CorrelationID *string                `json:"correlation_id,omitempty"`
	ReplyToID     *int64                 `json:"reply_to_id,omitempty"`
}

// MarkReadRequest represents a request to mark messages as read
type MarkReadRequest struct {
	MessageIDs []int64 `json:"message_ids" binding:"required"`
}

// SendMessage handles POST /messages
// @Summary Send a message to another pod
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body SendMessageRequest true "Message request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /messages [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req AgentSendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.msgSvc.SendMessage(
		c.Request.Context(),
		podKey,
		req.ReceiverPod,
		req.MessageType,
		agent.MessageContent(req.Content),
		req.CorrelationID,
		req.ReplyToID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	count, err := h.msgSvc.GetUnreadCount(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkRead handles POST /messages/mark-read
// @Summary Mark one or more messages as read
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Param request body MarkReadRequest true "Mark read request"
// @Success 200 {object} map[string]interface{}
// @Router /messages/mark-read [post]
func (h *MessageHandler) MarkRead(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	markedCount := 0
	for _, messageID := range req.MessageIDs {
		if err := h.msgSvc.MarkRead(c.Request.Context(), messageID, podKey); err == nil {
			markedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{"marked_count": markedCount})
}

// MarkAllRead handles POST /messages/mark-all-read
// @Summary Mark all messages for this pod as read
// @Tags messages
// @Accept json
// @Produce json
// @Param X-Pod-Key header string true "Pod Key"
// @Success 200 {object} map[string]interface{}
// @Router /messages/mark-all-read [post]
func (h *MessageHandler) MarkAllRead(c *gin.Context) {
	podKey := getPodKeyFromHeader(c)
	if podKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	count, err := h.msgSvc.MarkAllRead(c.Request.Context(), podKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_count": count})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	messageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message ID"})
		return
	}

	message, err := h.msgSvc.GetMessage(c.Request.Context(), messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}

	// Check access
	if message.SenderPod != podKey && message.ReceiverPod != podKey {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized to view this message"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
		return
	}

	correlationID := c.Param("correlation_id")
	if correlationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "correlation_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Pod-Key header required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    len(messages),
	})
}

// GetDeadLetters handles GET /messages/dlq
// @Summary Get dead letter queue entries
// @Tags messages
// @Accept json
// @Produce json
// @Param limit query int false "Maximum entries to return" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{}
// @Router /messages/dlq [get]
func (h *MessageHandler) GetDeadLetters(c *gin.Context) {
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

	entries, err := h.msgSvc.GetDeadLetters(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   len(entries),
	})
}

// ReplayDeadLetter handles POST /messages/dlq/:id/replay
// @Summary Replay a dead letter message
// @Tags messages
// @Accept json
// @Produce json
// @Param id path int true "Dead Letter Entry ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /messages/dlq/{id}/replay [post]
func (h *MessageHandler) ReplayDeadLetter(c *gin.Context) {
	entryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	message, err := h.msgSvc.ReplayDeadLetter(c.Request.Context(), entryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Replayed successfully",
		"replayed_message": message,
	})
}
