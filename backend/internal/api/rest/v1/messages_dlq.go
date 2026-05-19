package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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
