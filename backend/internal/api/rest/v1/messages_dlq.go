package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
		apierr.InternalError(c, err.Error())
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
		apierr.InvalidInput(c, "invalid entry ID")
		return
	}

	message, err := h.msgSvc.ReplayDeadLetter(c.Request.Context(), entryID)
	if err != nil {
		apierr.ResourceNotFound(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Replayed successfully",
		"replayed_message": message,
	})
}
