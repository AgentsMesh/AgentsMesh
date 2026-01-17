package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/gin-gonic/gin"
)

// ChannelHandler handles channel-related requests
type ChannelHandler struct {
	channelService *channel.Service
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(channelService *channel.Service) *ChannelHandler {
	return &ChannelHandler{
		channelService: channelService,
	}
}

// ListChannelsRequest represents channel list request
type ListChannelsRequest struct {
	RepositoryID    *int64 `form:"repository_id"`
	TicketID        *int64 `form:"ticket_id"`
	IncludeArchived bool   `form:"include_archived"`
}

// ListChannels lists channels
// GET /api/v1/organizations/:slug/channels
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	var req ListChannelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	limit := 50
	offset := 0

	channels, total, err := h.channelService.ListChannels(c.Request.Context(), tenant.OrganizationID, req.IncludeArchived, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels, "total": total})
}

// CreateChannelRequest represents channel creation request
type CreateChannelRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=100"`
	Description  string `json:"description"`
	Document     string `json:"document"`
	RepositoryID *int64 `json:"repository_id"`
	TicketID     *int64 `json:"ticket_id"`
}

// CreateChannel creates a new channel
// POST /api/v1/organizations/:slug/channels
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}

	// Check if request is from a Pod (via Pod API path)
	var createdByPod *string
	if pk, exists := c.Get("pod_key"); exists {
		if pkStr, ok := pk.(string); ok && pkStr != "" {
			createdByPod = &pkStr
		}
	}

	ch, err := h.channelService.CreateChannel(c.Request.Context(), &channel.CreateChannelRequest{
		OrganizationID:  tenant.OrganizationID,
		Name:            req.Name,
		Description:     desc,
		RepositoryID:    req.RepositoryID,
		TicketID:        req.TicketID,
		CreatedByPod:    createdByPod,
		CreatedByUserID: &tenant.UserID,
	})
	if err != nil {
		if err == channel.ErrDuplicateName {
			c.JSON(http.StatusConflict, gin.H{"error": "Channel name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"channel": ch})
}

// GetChannel returns channel by ID
// GET /api/v1/organizations/:slug/channels/:id
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channel": ch})
}

// ArchiveChannel archives a channel
// POST /api/v1/organizations/:slug/channels/:id/archive
func (h *ChannelHandler) ArchiveChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.channelService.ArchiveChannel(c.Request.Context(), channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel archived"})
}

// UnarchiveChannel unarchives a channel
// POST /api/v1/organizations/:slug/channels/:id/unarchive
func (h *ChannelHandler) UnarchiveChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.channelService.UnarchiveChannel(c.Request.Context(), channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unarchive channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel unarchived"})
}

// ListMessages lists channel messages
// GET /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) ListMessages(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	messages, err := h.channelService.GetMessages(c.Request.Context(), channelID, nil, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendMessageRequest represents message send request
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
	PodKey  string `json:"pod_key"`
}

// SendMessage sends a message to a channel
// POST /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) SendMessage(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if ch.IsArchived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send messages to archived channel"})
		return
	}

	// Determine sender pod key:
	// 1. If called via Pod API (/api/v1/orgs/:slug/pod/channels/*), use pod_key from context
	// 2. Otherwise, use pod_key from request body (for user-initiated messages)
	var podKey *string
	if pk, exists := c.Get("pod_key"); exists {
		// Pod API path - use the authenticated pod's key
		if pkStr, ok := pk.(string); ok && pkStr != "" {
			podKey = &pkStr
		}
	} else if req.PodKey != "" {
		// User API path with explicit pod_key in request
		podKey = &req.PodKey
	}

	msg, err := h.channelService.SendMessage(c.Request.Context(), channelID, podKey, &tenant.UserID, "text", req.Content, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": msg})
}

// JoinPod joins a pod to a channel
// POST /api/v1/organizations/:slug/channels/:id/pods
func (h *ChannelHandler) JoinPod(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req struct {
		PodKey string `json:"pod_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.channelService.JoinChannel(c.Request.Context(), channelID, req.PodKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod joined channel"})
}

// LeavePod removes a pod from a channel
// DELETE /api/v1/organizations/:slug/channels/:id/pods/:pod_key
func (h *ChannelHandler) LeavePod(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	podKey := c.Param("pod_key")

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.channelService.LeaveChannel(c.Request.Context(), channelID, podKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pod left channel"})
}

// ListChannelPods returns pods joined to a channel
// GET /api/v1/organizations/:slug/channels/:id/pods
func (h *ChannelHandler) ListChannelPods(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if ch.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	pods, err := h.channelService.GetChannelPods(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list pods"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pods":  pods,
		"total": len(pods),
	})
}

// GetDocument returns the channel document
// GET /api/v1/org/:slug/channels/:id/document
func (h *ChannelHandler) GetDocument(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	ch, err := h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	document := ""
	if ch.Document != nil {
		document = *ch.Document
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

// UpdateDocumentRequest represents document update request
type UpdateDocumentRequest struct {
	Document string `json:"document" binding:"required"`
}

// UpdateDocument updates the channel document
// PUT /api/v1/org/:slug/channels/:id/document
func (h *ChannelHandler) UpdateDocument(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var req UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	_, err = h.channelService.UpdateChannel(c.Request.Context(), channelID, nil, nil, &req.Document)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": req.Document})
}
