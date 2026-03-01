package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
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
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	count, err := h.msgSvc.MarkAllRead(c.Request.Context(), podKey)
	if err != nil {
		apierr.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"marked_count": count})
}
