package v1

import (
	"errors"
	"net/http"
	"strconv"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	channelService "github.com/anthropics/agentsmesh/backend/internal/service/channel"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ListMessages lists channel messages
// GET /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) ListMessages(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var messages []*channelDomain.Message
	var hasMore bool
	if beforeIDStr := c.Query("before_id"); beforeIDStr != "" {
		beforeID, err := strconv.ParseInt(beforeIDStr, 10, 64)
		if err != nil {
			apierr.InvalidInput(c, "Invalid before_id")
			return
		}
		messages, hasMore, err = h.channelService.GetMessagesByCursor(c.Request.Context(), ch.ID, beforeID, limit)
		if err != nil {
			apierr.InternalError(c, "Failed to list messages")
			return
		}
	} else {
		var fetchErr error
		messages, hasMore, fetchErr = h.channelService.GetMessages(c.Request.Context(), ch.ID, nil, nil, limit)
		if fetchErr != nil {
			apierr.InternalError(c, "Failed to list messages")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages, "has_more": hasMore})
}

// MentionRefRequest is the wire-format mapping a `@<key>` substring in a
// markdown source string to a typed entity reference. Used by Send/EditMessage
// when `source` is supplied.
type MentionRefRequest struct {
	EntityType string `json:"entity_type"`
	EntityKey  string `json:"entity_key"`
}

// SendMessageRequest accepts EITHER a markdown source string (server parses to
// AST) OR a pre-built MessageContent AST. Sending both is a 400.
type SendMessageRequest struct {
	Source        *string                       `json:"source,omitempty"`
	Mentions      map[string]MentionRefRequest  `json:"mentions,omitempty"`
	Content       *channelDomain.MessageContent `json:"content,omitempty"`
	AttachmentKey string                        `json:"attachment_key,omitempty"`
	PodKey        string                        `json:"pod_key"`
	ReplyTo       *int64                        `json:"reply_to"`
}

func resolveContent(source *string, mentions map[string]MentionRefRequest, content *channelDomain.MessageContent, attachmentKey string) (channelDomain.MessageContent, error) {
	hasSource := source != nil && *source != ""
	hasContent := content != nil && len(content.Blocks) > 0
	if hasSource && hasContent {
		return channelDomain.MessageContent{}, errors.New("provide either source or content, not both")
	}
	var resolved channelDomain.MessageContent
	switch {
	case hasSource:
		refs := make(map[string]channelService.MentionRef, len(mentions))
		for display, ref := range mentions {
			refs[display] = channelService.MentionRef{EntityType: ref.EntityType, EntityKey: ref.EntityKey}
		}
		c, err := channelService.ParseMarkdown(*source, refs)
		if err != nil {
			return channelDomain.MessageContent{}, err
		}
		resolved = c
	case hasContent:
		resolved = *content
	case attachmentKey != "":
		resolved = channelDomain.MessageContent{Kind: "text"}
	default:
		return channelDomain.MessageContent{}, errors.New("source, content, or attachment_key is required")
	}
	if attachmentKey != "" {
		resolved.AttachmentKey = attachmentKey
	}
	return resolved, nil
}

// SendMessage sends a message to a channel
// POST /api/v1/organizations/:slug/channels/:id/messages
func (h *ChannelHandler) SendMessage(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if ch.IsArchived {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot send messages to archived channel")
		return
	}

	content, err := resolveContent(req.Source, req.Mentions, req.Content, req.AttachmentKey)
	if err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	var podKey *string
	if req.PodKey != "" {
		podKey = &req.PodKey
	}

	msg, err := h.channelService.SendMessage(c.Request.Context(), ch.ID, podKey, &tenant.UserID, content, req.ReplyTo)
	if err != nil {
		if errors.Is(err, channelService.ErrNotMember) {
			apierr.ForbiddenAccess(c)
			return
		}
		if errors.Is(err, channelService.ErrEmptyContent) || errors.Is(err, channelService.ErrInvalidContent) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, err.Error())
			return
		}
		apierr.InternalError(c, "Failed to send message")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": msg})
}

// EditMessageRequest mirrors SendMessageRequest's source/content choice.
type EditMessageRequest struct {
	Source        *string                       `json:"source,omitempty"`
	Mentions      map[string]MentionRefRequest  `json:"mentions,omitempty"`
	Content       *channelDomain.MessageContent `json:"content,omitempty"`
	AttachmentKey string                        `json:"attachment_key,omitempty"`
}

// EditMessage edits a channel message
// PUT /api/v1/organizations/:slug/channels/:id/messages/:msg_id
func (h *ChannelHandler) EditMessage(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	msgID, err := strconv.ParseInt(c.Param("msg_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid message ID")
		return
	}

	var req EditMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	content, err := resolveContent(req.Source, req.Mentions, req.Content, req.AttachmentKey)
	if err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	msg, err := h.channelService.EditMessage(c.Request.Context(), ch.ID, msgID, tenant.UserID, content)
	if err != nil {
		switch {
		case errors.Is(err, channelService.ErrMessageNotFound):
			apierr.ResourceNotFound(c, "Message not found")
		case errors.Is(err, channelService.ErrNotMessageSender):
			apierr.ForbiddenAccess(c)
		case errors.Is(err, channelService.ErrChannelArchived):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot edit messages in archived channel")
		case errors.Is(err, channelService.ErrEmptyContent), errors.Is(err, channelService.ErrInvalidContent):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, err.Error())
		default:
			apierr.InternalError(c, "Failed to edit message")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// DeleteMessage soft-deletes a channel message
// DELETE /api/v1/organizations/:slug/channels/:id/messages/:msg_id
func (h *ChannelHandler) DeleteMessage(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	msgID, err := strconv.ParseInt(c.Param("msg_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid message ID")
		return
	}

	tenant := middleware.GetTenant(c)
	err = h.channelService.DeleteMessage(c.Request.Context(), ch.ID, msgID, tenant.UserID)
	if err != nil {
		switch {
		case errors.Is(err, channelService.ErrMessageNotFound):
			apierr.ResourceNotFound(c, "Message not found")
		case errors.Is(err, channelService.ErrNotMessageSender):
			apierr.ForbiddenAccess(c)
		case errors.Is(err, channelService.ErrChannelArchived):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot delete messages in archived channel")
		default:
			apierr.InternalError(c, "Failed to delete message")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// SearchMessages searches channel messages by full-text query
// GET /api/v1/organizations/:slug/channels/:id/messages/search?q=term&limit=20
func (h *ChannelHandler) SearchMessages(c *gin.Context) {
	ch, ok := h.requireChannelAccess(c)
	if !ok {
		return
	}

	query := c.Query("q")
	if query == "" {
		apierr.InvalidInput(c, "Search query is required")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	messages, err := h.channelService.SearchMessages(c.Request.Context(), ch.ID, query, limit)
	if err != nil {
		apierr.InternalError(c, "Failed to search messages")
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
