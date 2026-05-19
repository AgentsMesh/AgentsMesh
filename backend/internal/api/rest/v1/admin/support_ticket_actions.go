package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *SupportTicketHandler) Reply(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	adminUserID := middleware.GetAdminUserID(c)

	content := c.PostForm("content")
	if content == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Content is required")
		return
	}

	msg, err := h.service.AdminAddReply(c.Request.Context(), ticketID, adminUserID, &supportticket.AddMessageRequest{
		Content: content,
	})
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to add reply")
		return
	}

	uploadReplyAttachments(c, h.service, ticketID, adminUserID, msg.ID)

	h.logAction(c, admin.AuditActionSupportTicketReply, admin.TargetTypeSupportTicket, ticketID, nil, gin.H{"content": content})

	c.JSON(http.StatusCreated, msg)
}

func uploadReplyAttachments(c *gin.Context, svc *supportticket.Service, ticketID, adminUserID, msgID int64) {
	form, _ := c.MultipartForm()
	if form == nil || form.File["files[]"] == nil {
		return
	}
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
			if _, err := svc.UploadAttachment(c.Request.Context(), ticketID, adminUserID, &msgID, true, &supportticket.UploadAttachmentRequest{
				FileName:    fileHeader.Filename,
				ContentType: contentType,
				Size:        fileHeader.Size,
				Reader:      file,
			}); err != nil {
				slog.WarnContext(c.Request.Context(), "failed to upload admin attachment", "filename", fileHeader.Filename, "error", err)
			}
		}()
	}
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *SupportTicketHandler) UpdateStatus(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	oldTicket, err := h.service.AdminGetByID(c.Request.Context(), ticketID)
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to get support ticket")
		return
	}

	if err := h.service.AdminUpdateStatus(c.Request.Context(), ticketID, req.Status); err != nil {
		handleStatusUpdateError(c, err)
		return
	}

	h.logAction(c, admin.AuditActionSupportTicketStatus, admin.TargetTypeSupportTicket, ticketID,
		gin.H{"status": oldTicket.Status}, gin.H{"status": req.Status})

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

func handleStatusUpdateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, supportticket.ErrInvalidStatus):
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid status")
	case errors.Is(err, supportticket.ErrInvalidTransition):
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid status transition")
	case errors.Is(err, supportticket.ErrTicketNotFound):
		apierr.ResourceNotFound(c, "Support ticket not found")
	default:
		apierr.InternalError(c, "Failed to update status")
	}
}

type AssignRequest struct {
	AdminID *int64 `json:"admin_id"`
}

func (h *SupportTicketHandler) Assign(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	adminUserID := middleware.GetAdminUserID(c)

	var req AssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.AdminID = &adminUserID
	}
	if req.AdminID == nil {
		req.AdminID = &adminUserID
	}

	if err := h.service.AdminAssign(c.Request.Context(), ticketID, *req.AdminID); err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to assign ticket")
		return
	}

	h.logAction(c, admin.AuditActionSupportTicketAssign, admin.TargetTypeSupportTicket, ticketID,
		nil, gin.H{"assigned_admin_id": *req.AdminID})

	c.JSON(http.StatusOK, gin.H{"message": "Ticket assigned"})
}

func (h *SupportTicketHandler) GetAttachmentURL(c *gin.Context) {
	attachmentID, err := strconv.ParseInt(c.Param("attachmentId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid attachment ID")
		return
	}

	url, err := h.service.AdminGetAttachmentURL(c.Request.Context(), attachmentID)
	if err != nil {
		if errors.Is(err, supportticket.ErrAttachmentNotFound) {
			apierr.ResourceNotFound(c, "Attachment not found")
			return
		}
		apierr.InternalError(c, "Failed to get attachment URL")
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}
