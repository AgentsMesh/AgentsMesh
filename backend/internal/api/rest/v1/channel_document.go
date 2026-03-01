package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// GetDocument returns the channel document
// GET /api/v1/org/:slug/channels/:id/document
func (h *ChannelHandler) GetDocument(c *gin.Context) {
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
		apierr.InvalidInput(c, "Invalid channel ID")
		return
	}

	var req UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	_, err = h.channelService.GetChannel(c.Request.Context(), channelID)
	if err != nil {
		apierr.ResourceNotFound(c, "Channel not found")
		return
	}

	_, err = h.channelService.UpdateChannel(c.Request.Context(), channelID, nil, nil, &req.Document)
	if err != nil {
		apierr.InternalError(c, "Failed to update document")
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": req.Document})
}
