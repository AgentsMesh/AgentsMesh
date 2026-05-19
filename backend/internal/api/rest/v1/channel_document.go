package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *ChannelHandler) GetDocument(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	document := ""
	if ch.Document != nil {
		document = *ch.Document
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

type UpdateDocumentRequest struct {
	Document string `json:"document" binding:"required"`
}

func (h *ChannelHandler) UpdateDocument(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	_, err := h.channelService.UpdateChannel(c.Request.Context(), ch.ID, nil, nil, &req.Document)
	if err != nil {
		apierr.InternalError(c, "Failed to update document")
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": req.Document})
}
