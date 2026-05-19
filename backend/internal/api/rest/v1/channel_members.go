package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *ChannelHandler) ListMembers(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
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

	members, total, err := h.channelService.ListMembers(c.Request.Context(), ch.ID, limit, offset)
	if err != nil {
		apierr.InternalError(c, "Failed to list members")
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members, "total": total})
}
