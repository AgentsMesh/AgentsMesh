package ticket

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ========== Errors ==========

var (
	ErrTicketNotFound    = errors.New("ticket not found")
	ErrLabelNotFound     = errors.New("label not found")
	ErrDuplicateLabel    = errors.New("label already exists")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// ========== Service ==========

// Service handles ticket operations.
type Service struct {
	repo           ticket.TicketRepository
	eventPublisher EventPublisher
}

// NewService creates a new ticket service.
func NewService(repo ticket.TicketRepository) *Service {
	return &Service{repo: repo}
}

// SetEventPublisher sets the event publisher for real-time events.
func (s *Service) SetEventPublisher(ep EventPublisher) {
	s.eventPublisher = ep
}

// publishEvent publishes a ticket event if EventPublisher is configured.
func (s *Service) publishEvent(ctx context.Context, eventType TicketEventType, orgID int64, slug, status, previousStatus string) {
	if s.eventPublisher != nil {
		s.eventPublisher.PublishTicketEvent(ctx, eventType, orgID, slug, status, previousStatus)
	}
}

// ========== Request Types ==========

// CreateTicketRequest represents a ticket creation request.
type CreateTicketRequest struct {
	OrganizationID int64
	RepositoryID   *int64
	ReporterID     int64
	ParentTicketID *int64
	Title          string
	Content        *string
	Status         string
	Priority       string
	DueDate        *time.Time
	AssigneeIDs    []int64
	LabelIDs       []int64
	Labels         []string // Label names for convenience
}
