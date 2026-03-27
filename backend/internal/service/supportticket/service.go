package supportticket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"path"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/google/uuid"
)

var (
	ErrTicketNotFound     = errors.New("support ticket not found")
	ErrAccessDenied       = errors.New("access denied")
	ErrInvalidCategory    = errors.New("invalid category")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrInvalidPriority    = errors.New("invalid priority")
	ErrStorageError       = errors.New("storage operation failed")
	ErrFileTooLarge       = errors.New("file exceeds maximum size")
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// Service handles support ticket operations
type Service struct {
	repo    supportticket.Repository
	storage storage.Storage
	config  config.StorageConfig
}

// NewService creates a new support ticket service
func NewService(repo supportticket.Repository, storage storage.Storage, cfg config.StorageConfig) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		config:  cfg,
	}
}

// --- Request/Response types ---

// CreateRequest represents a request to create a support ticket
type CreateRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Content  string `json:"content"`
	Priority string `json:"priority"`
}

// AddMessageRequest represents a request to add a message to a ticket
type AddMessageRequest struct {
	Content string `json:"content"`
}

// UploadAttachmentRequest represents a file upload for a ticket attachment
type UploadAttachmentRequest struct {
	FileName    string
	ContentType string
	Size        int64
	Reader      io.Reader
}

// ListQuery represents query parameters for listing user tickets
type ListQuery struct {
	Status   string
	Page     int
	PageSize int
}

// AdminListQuery represents query parameters for admin listing
type AdminListQuery struct {
	Search   string
	Status   string
	Category string
	Priority string
	Page     int
	PageSize int
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Data       []supportticket.SupportTicket `json:"data"`
	Total      int64                         `json:"total"`
	Page       int                           `json:"page"`
	PageSize   int                           `json:"page_size"`
	TotalPages int                           `json:"total_pages"`
}

// Stats represents support ticket statistics
type Stats struct {
	Total      int64 `json:"total"`
	Open       int64 `json:"open"`
	InProgress int64 `json:"in_progress"`
	Resolved   int64 `json:"resolved"`
	Closed     int64 `json:"closed"`
}

// --- User-side methods ---

// Create creates a new support ticket with an initial message
func (s *Service) Create(ctx context.Context, userID int64, req *CreateRequest) (*supportticket.SupportTicket, error) {
	category := req.Category
	if category == "" {
		category = supportticket.CategoryOther
	}
	if !supportticket.ValidCategories[category] {
		return nil, ErrInvalidCategory
	}

	priority := req.Priority
	if priority == "" {
		priority = supportticket.PriorityMedium
	}
	if !supportticket.ValidPriorities[priority] {
		return nil, ErrInvalidPriority
	}

	ticket := supportticket.SupportTicket{
		UserID:   userID,
		Title:    req.Title,
		Category: category,
		Status:   supportticket.StatusOpen,
		Priority: priority,
	}

	var msg *supportticket.SupportTicketMessage
	if req.Content != "" {
		msg = &supportticket.SupportTicketMessage{
			UserID:       userID,
			Content:      req.Content,
			IsAdminReply: false,
		}
	}

	if err := s.repo.CreateTicketWithMessage(ctx, &ticket, msg); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	return &ticket, nil
}

// ListByUser returns paginated tickets for a specific user
func (s *Service) ListByUser(ctx context.Context, userID int64, query *ListQuery) (*ListResponse, error) {
	page, pageSize := normalizePagination(query.Page, query.PageSize)
	offset := (page - 1) * pageSize

	tickets, total, err := s.repo.ListByUser(ctx, userID, query.Status, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	return &ListResponse{
		Data:       tickets,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

// GetByID returns a ticket by ID, verifying user ownership
func (s *Service) GetByID(ctx context.Context, id, userID int64) (*supportticket.SupportTicket, error) {
	ticket, err := s.repo.GetByIDAndUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	return ticket, nil
}

// AddMessage adds a user message to a ticket
func (s *Service) AddMessage(ctx context.Context, ticketID, userID int64, req *AddMessageRequest) (*supportticket.SupportTicketMessage, error) {
	if _, err := s.GetByID(ctx, ticketID, userID); err != nil {
		return nil, err
	}

	msg := &supportticket.SupportTicketMessage{
		TicketID:     ticketID,
		UserID:       userID,
		Content:      req.Content,
		IsAdminReply: false,
	}

	if err := s.repo.AddMessageAndReopen(ctx, msg, ticketID); err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}
	return msg, nil
}

// ListMessages returns all messages for a ticket (user-side, verifies ownership)
func (s *Service) ListMessages(ctx context.Context, ticketID, userID int64) ([]supportticket.SupportTicketMessage, error) {
	if _, err := s.GetByID(ctx, ticketID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListMessagesByTicketID(ctx, ticketID)
}

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

// --- Internal helpers ---

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
