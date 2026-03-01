package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "X-Pod-Key header required")
		return
	}

	var req AgentSendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
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
		apierr.ValidationError(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}
