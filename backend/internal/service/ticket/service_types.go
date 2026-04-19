package ticket

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
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
	// blockstore is optional. When set, ticket content writes are stored as
	// Block Store `document` blocks (ticket row carries only the block id).
	// Legacy code paths that still populate Ticket.Content directly remain
	// valid when blockstore is nil, easing staged rollout. No DB-level FK:
	// cross-aggregate lifecycle (delete ticket → delete block) is enforced
	// in this service layer; a stale content_block_id reads gracefully to
	// empty on the consumer side.
	blockstore *blockstoreservice.Service
}

// NewService creates a new ticket service.
func NewService(repo ticket.TicketRepository) *Service {
	return &Service{repo: repo}
}

// SetBlockstore wires the Block Store service. When nil, ticket content
// continues to live in the legacy tickets.content column.
func (s *Service) SetBlockstore(bs *blockstoreservice.Service) {
	s.blockstore = bs
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
