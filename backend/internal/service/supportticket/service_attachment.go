package supportticket

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"github.com/google/uuid"
)

// UploadAttachment uploads a file attachment and associates it with a ticket/message
func (s *Service) UploadAttachment(ctx context.Context, ticketID, userID int64, messageID *int64, isAdmin bool, req *UploadAttachmentRequest) (*supportticket.SupportTicketAttachment, error) {
	if s.storage == nil {
		return nil, ErrStorageError
	}

	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	if ticket.UserID != userID && !isAdmin {
		return nil, ErrAccessDenied
	}

	maxSize := s.config.MaxFileSize * 1024 * 1024
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}
	if req.Size > maxSize {
		return nil, ErrFileTooLarge
	}

	ext := path.Ext(req.FileName)
	if ext == "" {
		ext = ".bin"
	}
	now := time.Now()
	storageKey := fmt.Sprintf("support-tickets/%d/%d/%02d/%s%s",
		userID, now.Year(), now.Month(), uuid.New().String(), ext)

	if _, err := s.storage.Upload(ctx, storageKey, req.Reader, req.Size, req.ContentType); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageError, err)
	}

	attachment := &supportticket.SupportTicketAttachment{
		TicketID:     ticketID,
		MessageID:    messageID,
		UploaderID:   userID,
		OriginalName: req.FileName,
		StorageKey:   storageKey,
		MimeType:     req.ContentType,
		Size:         req.Size,
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		if delErr := s.storage.Delete(ctx, storageKey); delErr != nil {
			slog.Warn("failed to cleanup uploaded file after DB error", "storage_key", storageKey, "error", delErr)
		}
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	return attachment, nil
}

// GetAttachmentURL returns a presigned URL for downloading an attachment
func (s *Service) GetAttachmentURL(ctx context.Context, attachmentID, userID int64) (string, error) {
	if s.storage == nil {
		return "", ErrStorageError
	}

	attachment, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	if attachment == nil {
		return "", ErrAttachmentNotFound
	}

	ticket, err := s.repo.GetTicketByID(ctx, attachment.TicketID)
	if err != nil {
		return "", err
	}
	if ticket == nil {
		return "", ErrTicketNotFound
	}
	if ticket.UserID != userID {
		return "", ErrAccessDenied
	}

	return s.storage.GetURL(ctx, attachment.StorageKey, 1*time.Hour)
}
