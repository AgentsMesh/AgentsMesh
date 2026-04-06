package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ChannelHandler handles channel-related requests
type ChannelHandler struct {
	channelService *channel.Service
	ticketService  *ticket.Service
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelService *channel.Service, ticketService *ticket.Service) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		ticketService:  ticketService,
	}
}

// ListChannelsRequest represents channel list request
type ListChannelsRequest struct {
	RepositoryID    *int64  `form:"repository_id"`
	TicketSlug      *string `form:"ticket_slug"`
	IncludeArchived bool    `form:"include_archived"`
}

// ListChannels lists channels
// GET /api/v1/organizations/:slug/channels
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	var req ListChannelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	ctx := c.Request.Context()

	// Resolve ticket slug to ID for filtering
	var ticketID *int64
	if req.TicketSlug != nil && *req.TicketSlug != "" {
		t, err := h.ticketService.GetTicketByIDOrSlug(ctx, tenant.OrganizationID, *req.TicketSlug)
		if err != nil {
			// If ticket not found, return empty results
			c.JSON(http.StatusOK, gin.H{"channels": []interface{}{}, "total": 0})
			return
		}
		ticketID = &t.ID
	}

	channels, total, err := h.channelService.ListChannels(ctx, tenant.OrganizationID, &channelDomain.ChannelListFilter{
		IncludeArchived: req.IncludeArchived,
		RepositoryID:    req.RepositoryID,
		TicketID:        ticketID,
		Limit:           50,
		Offset:          0,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list channels")
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels, "total": total})
}

// CreateChannelRequest represents channel creation request
type CreateChannelRequest struct {
	Name         string  `json:"name" binding:"required,min=2,max=100"`
	Description  string  `json:"description"`
	Document     string  `json:"document"`
	RepositoryID *int64  `json:"repository_id"`
	TicketSlug   *string `json:"ticket_slug"`
}

// CreateChannel creates a new channel
// POST /api/v1/organizations/:slug/channels
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	// Resolve ticket slug to ID
	var ticketID *int64
	if req.TicketSlug != nil && *req.TicketSlug != "" {
		t, err := h.ticketService.GetTicketByIDOrSlug(c.Request.Context(), tenant.OrganizationID, *req.TicketSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
			return
		}
		ticketID = &t.ID
	}

	ch, err := h.channelService.CreateChannel(c.Request.Context(), &channel.CreateChannelRequest{
		OrganizationID:  tenant.OrganizationID,
		Name:            req.Name,
		Description:     desc,
		RepositoryID:    req.RepositoryID,
		TicketID:        ticketID,
		CreatedByUserID: &tenant.UserID,
	})
	if err != nil {
		if err == channel.ErrDuplicateName {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Channel name already exists")
			return
		}
		apierr.InternalError(c, "Failed to create channel")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"channel": ch})
}

// GetChannel returns channel by ID
// GET /api/v1/organizations/:slug/channels/:id
func (h *ChannelHandler) GetChannel(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"channel": ch})
}

// UpdateChannelRequest represents channel update request
type UpdateChannelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Document    string `json:"document"`
}

// UpdateChannel updates a channel
// PUT /api/v1/organizations/:slug/channels/:id
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
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

	var name, description, document *string
	if req.Name != "" {
		name = &req.Name
	}
	if req.Description != "" {
		description = &req.Description
	}
	if req.Document != "" {
		document = &req.Document
	}

	ch, err = h.channelService.UpdateChannel(c.Request.Context(), channelID, name, description, document)
	if err != nil {
		apierr.InternalError(c, "Failed to update channel")
		return
	}

	c.JSON(http.StatusOK, gin.H{"channel": ch})
}
