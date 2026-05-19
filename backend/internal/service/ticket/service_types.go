package ticket

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
)

var (
	ErrTicketNotFound    = errors.New("ticket not found")
	ErrLabelNotFound     = errors.New("label not found")
	ErrDuplicateLabel    = errors.New("label already exists")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type Service struct {
	repo           ticket.TicketRepository
	eventPublisher EventPublisher
	blockstore *blockstoreservice.Service
}

func NewService(repo ticket.TicketRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetBlockstore(bs *blockstoreservice.Service) {
	s.blockstore = bs
}

func (s *Service) SetEventPublisher(ep EventPublisher) {
	s.eventPublisher = ep
}

func (s *Service) publishEvent(ctx context.Context, eventType TicketEventType, orgID int64, slug, status, previousStatus string) {
	if s.eventPublisher != nil {
		s.eventPublisher.PublishTicketEvent(ctx, eventType, orgID, slug, status, previousStatus)
	}
}

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
