package ticket

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ========== Errors ==========

var (
	ErrTicketNotFound    = errors.New("ticket not found")
	ErrLabelNotFound     = errors.New("label not found")
	ErrDuplicateLabel    = errors.New("label already exists")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// ========== Service ==========

// Service handles ticket operations
type Service struct {
	db             *gorm.DB
	eventPublisher EventPublisher
}

// NewService creates a new ticket service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SetEventPublisher sets the event publisher for real-time events
func (s *Service) SetEventPublisher(ep EventPublisher) {
	s.eventPublisher = ep
}

// publishEvent publishes a ticket event if EventPublisher is configured
func (s *Service) publishEvent(ctx context.Context, eventType TicketEventType, orgID int64, identifier, status, previousStatus string) {
	if s.eventPublisher != nil {
		s.eventPublisher.PublishTicketEvent(ctx, eventType, orgID, identifier, status, previousStatus)
	}
}

// ========== Request Types ==========

// CreateTicketRequest represents a ticket creation request
type CreateTicketRequest struct {
	OrganizationID int64
	RepositoryID   *int64
	ReporterID     int64
	ParentTicketID *int64
	Type           string
	Title          string
	Content        *string
	Status         string
	Priority       string
	DueDate        *time.Time
	AssigneeIDs    []int64
	LabelIDs       []int64
	Labels         []string // Label names for convenience
}

// ListTicketsFilter represents filters for listing tickets
type ListTicketsFilter struct {
	OrganizationID int64
	RepositoryID   *int64
	Status         string
	Type           string
	Priority       string
	AssigneeID     *int64
	ReporterID     *int64
	LabelIDs       []int64
	ParentTicketID *int64
	Query          string
	UserRole       string // Kept for future use
	Limit          int
	Offset         int
}
