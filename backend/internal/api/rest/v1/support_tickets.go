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

type SupportTicketHandler struct {
	service *supportticket.Service
}

func NewSupportTicketHandler(service *supportticket.Service) *SupportTicketHandler {
	return &SupportTicketHandler{service: service}
}

func (h *SupportTicketHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.Create)
	rg.GET("", h.List)
	rg.GET("/:id", h.GetByID)
	rg.POST("/:id/messages", h.AddMessage)
	rg.GET("/:id/messages", h.ListMessages)
	rg.GET("/attachments/:attachmentId/url", h.GetAttachmentURL)
}

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

func (h *SupportTicketHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
		return
	}

	query := &supportticket.ListQuery{
		Status:   c.Query("status"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}

	result, err := h.service.ListByUser(c.Request.Context(), userID, query)
	if err != nil {
		apierr.InternalError(c, "Failed to list support tickets")
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *SupportTicketHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	ticket, err := h.service.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to get support ticket")
		return
	}

	messages, err := h.service.ListMessages(c.Request.Context(), id, userID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "failed to load messages for ticket", "ticket_id", id, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket":   ticket,
		"messages": messages,
	})
}

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

func (h *SupportTicketHandler) ListMessages(c *gin.Context) {
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

	messages, err := h.service.ListMessages(c.Request.Context(), ticketID, userID)
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to list messages")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

func (h *SupportTicketHandler) GetAttachmentURL(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authentication required")
		return
	}

	attachmentID, err := strconv.ParseInt(c.Param("attachmentId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid attachment ID")
		return
	}

	url, err := h.service.GetAttachmentURL(c.Request.Context(), attachmentID, userID)
	if err != nil {
		switch {
		case errors.Is(err, supportticket.ErrAttachmentNotFound):
			apierr.ResourceNotFound(c, "Attachment not found")
		case errors.Is(err, supportticket.ErrAccessDenied):
			apierr.ForbiddenAccess(c)
		default:
			apierr.InternalError(c, "Failed to get attachment URL")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
