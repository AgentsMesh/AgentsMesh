package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	channelService *channel.Service
	ticketService  *ticket.Service
}

func NewChannelHandler(channelService *channel.Service, ticketService *ticket.Service) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
		ticketService:  ticketService,
	}
}

type ListChannelsRequest struct {
	RepositoryID    *int64  `form:"repository_id"`
	TicketSlug      *string `form:"ticket_slug"`
	IncludeArchived bool    `form:"include_archived"`
}

func (h *ChannelHandler) ListChannels(c *gin.Context) {
	var req ListChannelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	ctx := c.Request.Context()

	var ticketID *int64
	if req.TicketSlug != nil && *req.TicketSlug != "" {
		t, err := h.ticketService.GetTicketByIDOrSlug(ctx, tenant.OrganizationID, *req.TicketSlug)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"channels": []interface{}{}, "total": 0})
			return
		}
		ticketID = &t.ID
	}

	channels, total, err := h.channelService.ListChannels(ctx, tenant.OrganizationID, tenant.UserID, &channelDomain.ChannelListFilter{
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

type CreateChannelRequest struct {
	Name         string  `json:"name" binding:"required,min=2,max=100"`
	Description  string  `json:"description"`
	Document     string  `json:"document"`
	RepositoryID *int64  `json:"repository_id"`
	TicketSlug   *string `json:"ticket_slug"`
	Visibility   string  `json:"visibility"`
	MemberIDs    []int64 `json:"member_ids"`
}

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
		OrganizationID:   tenant.OrganizationID,
		Name:             req.Name,
		Description:      desc,
		RepositoryID:     req.RepositoryID,
		TicketID:         ticketID,
		CreatedByUserID:  &tenant.UserID,
		Visibility:       req.Visibility,
		InitialMemberIDs: req.MemberIDs,
	})
	if err != nil {
		if errors.Is(err, channel.ErrDuplicateName) {
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Channel name already exists")
			return
		}
		apierr.InternalError(c, "Failed to create channel")
		return
	}

	enriched, err := h.channelService.GetChannelForUser(c.Request.Context(), ch.ID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{"channel": ch})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"channel": enriched})
}

func (h *ChannelHandler) GetChannel(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}
	tenant := middleware.GetTenant(c)
	enriched, err := h.channelService.GetChannelForUser(c.Request.Context(), ch.ID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"channel": ch})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channel": enriched})
}

type UpdateChannelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Document    string `json:"document"`
}

func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
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

	updated, err := h.channelService.UpdateChannel(c.Request.Context(), ch.ID, name, description, document)
	if err != nil {
		apierr.InternalError(c, "Failed to update channel")
		return
	}

	tenant := middleware.GetTenant(c)
	enriched, err := h.channelService.GetChannelForUser(c.Request.Context(), updated.ID, tenant.UserID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"channel": updated})
		return
	}
	c.JSON(http.StatusOK, gin.H{"channel": enriched})
}
