package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListMembers returns members of a channel with pagination
// GET /api/v1/organizations/:slug/channels/:id/members?limit=50&offset=0
func (h *ChannelHandler) ListMembers(c *gin.Context) {
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

	members, total, err := h.channelService.ListMembers(c.Request.Context(), channelID, limit, offset)
	if err != nil {
		apierr.InternalError(c, "Failed to list members")
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members, "total": total})
}
