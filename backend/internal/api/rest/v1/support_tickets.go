package v1

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SupportTicketHandler keeps the two multipart-bodied REST endpoints that
// Connect-RPC cannot represent today (file uploads). Read paths have moved
// to proto.support_ticket.v1.SupportTicketService (Connect).
type SupportTicketHandler struct {
	service *supportticket.Service
}

func NewSupportTicketHandler(service *supportticket.Service) *SupportTicketHandler {
	return &SupportTicketHandler{service: service}
}

// RegisterRoutes wires the two multipart endpoints. Connect-RPC owns
// list/get/messages-list/attachment-url.
func (h *SupportTicketHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.Create)
	rg.POST("/:id/messages", h.AddMessage)
}

// Create handles POST /api/v1/support-tickets (multipart/form-data).
// Kept on REST because Connect has no multipart wire today.
func (h *SupportTicketHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
		return
	}

	title := c.PostForm("title")
	if title == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Title is required")
		return
	}

	content := c.PostForm("content")
	if content == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Content is required")
		return
	}

	req := &supportticket.CreateRequest{
		Title:    title,
		Category: c.PostForm("category"),
		Content:  content,
		Priority: c.PostForm("priority"),
	}

	ticket, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, supportticket.ErrInvalidCategory):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid category")
		case errors.Is(err, supportticket.ErrInvalidPriority):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid priority")
		default:
			apierr.InternalError(c, "Failed to create support ticket")
		}
		return
	}

	form, _ := c.MultipartForm()
	if form != nil && form.File["files[]"] != nil {
		for _, fileHeader := range form.File["files[]"] {
			func() {
				file, err := fileHeader.Open()
				if err != nil {
					slog.WarnContext(c.Request.Context(), "failed to open uploaded file", "filename", fileHeader.Filename, "error", err)
					return
				}
				defer file.Close()
				contentType := fileHeader.Header.Get("Content-Type")
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				if _, err := h.service.UploadAttachment(c.Request.Context(), ticket.ID, userID, nil, false, &supportticket.UploadAttachmentRequest{
					FileName:    fileHeader.Filename,
					ContentType: contentType,
					Size:        fileHeader.Size,
					Reader:      file,
				}); err != nil {
					slog.WarnContext(c.Request.Context(), "failed to upload attachment", "filename", fileHeader.Filename, "error", err)
				}
			}()
		}
	}

	c.JSON(http.StatusCreated, ticket)
}

// AddMessage handles POST /api/v1/support-tickets/:id/messages
// (multipart/form-data). Kept on REST for the same reason as Create.
func (h *SupportTicketHandler) AddMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
		return
	}

	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	content := c.PostForm("content")
	if content == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Content is required")
		return
	}

	msg, err := h.service.AddMessage(c.Request.Context(), ticketID, userID, &supportticket.AddMessageRequest{
		Content: content,
	})
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to add message")
		return
	}

	form, _ := c.MultipartForm()
	if form != nil && form.File["files[]"] != nil {
		for _, fileHeader := range form.File["files[]"] {
			func() {
				file, err := fileHeader.Open()
				if err != nil {
					slog.WarnContext(c.Request.Context(), "failed to open uploaded file", "filename", fileHeader.Filename, "error", err)
					return
				}
				defer file.Close()
				contentType := fileHeader.Header.Get("Content-Type")
				if contentType == "" {
					contentType = "application/octet-stream"
				}
				if _, err := h.service.UploadAttachment(c.Request.Context(), ticketID, userID, &msg.ID, false, &supportticket.UploadAttachmentRequest{
					FileName:    fileHeader.Filename,
					ContentType: contentType,
					Size:        fileHeader.Size,
					Reader:      file,
				}); err != nil {
					slog.WarnContext(c.Request.Context(), "failed to upload attachment", "filename", fileHeader.Filename, "error", err)
				}
			}()
		}
	}

	c.JSON(http.StatusCreated, msg)
}
