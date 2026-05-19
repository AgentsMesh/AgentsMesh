package v1

import (
	"errors"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *ChannelHandler) requireChannelAccess(c *gin.Context) (*channel.Channel, bool) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return nil, false
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		if errors.Is(err, channelService.ErrChannelNotFound) {
			apierr.ResourceNotFound(c, "Channel not found")
		} else {
			apierr.InternalError(c, "Failed to get channel")
		}
		return nil, false
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		apierr.ForbiddenAccess(c)
		return nil, false
	}

	if !ch.IsPublic() {
		ok, err := h.channelService.IsMember(c.Request.Context(), ch.ID, tenant.UserID)
		if err != nil {
			apierr.InternalError(c, "Failed to check membership")
			return nil, false
		}
		if !ok {
			apierr.ForbiddenAccess(c)
			return nil, false
		}
	}

	return ch, true
}

func handleChannelServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, channelService.ErrNotMember):
		apierr.ForbiddenAccess(c)
	case errors.Is(err, channelService.ErrChannelPrivate):
		apierr.ForbiddenAccess(c)
	case errors.Is(err, channelService.ErrNotCreator):
		apierr.ForbiddenAccess(c)
	case errors.Is(err, channelService.ErrChannelArchived):
		apierr.Conflict(c, apierr.ALREADY_EXISTS, "Channel is archived")
	case errors.Is(err, channelService.ErrChannelNotFound):
		apierr.ResourceNotFound(c, "Channel not found")
	default:
		apierr.InternalError(c, "Operation failed")
	}
}
