package supportticket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/supportticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
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

type Service struct {
	repo    supportticket.Repository
	storage storage.Storage
	config  config.StorageConfig
}

func NewService(repo supportticket.Repository, storage storage.Storage, cfg config.StorageConfig) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		config:  cfg,
	}
}

type CreateRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Content  string `json:"content"`
	Priority string `json:"priority"`
}

type AddMessageRequest struct {
	Content string `json:"content"`
}

type UploadAttachmentRequest struct {
	FileName    string
	ContentType string
	Size        int64
	Reader      io.Reader
}

type ListQuery struct {
	Status   string
	Page     int
	PageSize int
}

type AdminListQuery struct {
	Search   string
	Status   string
	Category string
	Priority string
	Page     int
	PageSize int
}

type ListResponse struct {
	Data       []supportticket.SupportTicket `json:"data"`
	Total      int64                         `json:"total"`
	Page       int                           `json:"page"`
	PageSize   int                           `json:"page_size"`
	TotalPages int                           `json:"total_pages"`
}

type Stats struct {
	Total      int64 `json:"total"`
	Open       int64 `json:"open"`
	InProgress int64 `json:"in_progress"`
	Resolved   int64 `json:"resolved"`
	Closed     int64 `json:"closed"`
}

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
		slog.ErrorContext(ctx, "failed to create support ticket", "user_id", userID, "category", category, "error", err)
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	slog.InfoContext(ctx, "support ticket created", "ticket_id", ticket.ID, "user_id", userID, "category", category)
	return &ticket, nil
}

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
		slog.ErrorContext(ctx, "failed to add support ticket message", "ticket_id", ticketID, "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to add message: %w", err)
	}
	slog.InfoContext(ctx, "support ticket message added", "ticket_id", ticketID, "user_id", userID)
	return msg, nil
}

func (s *Service) ListMessages(ctx context.Context, ticketID, userID int64) ([]supportticket.SupportTicketMessage, error) {
	if _, err := s.GetByID(ctx, ticketID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListMessagesByTicketID(ctx, ticketID)
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
