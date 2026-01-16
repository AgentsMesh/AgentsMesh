package ticket

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
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
	Description    *string
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

// ========== Ticket CRUD ==========

// CreateTicket creates a new ticket
func (s *Service) CreateTicket(ctx context.Context, req *CreateTicketRequest) (*ticket.Ticket, error) {
	var number int
	var identifier string

	// Check if repository has a ticket_prefix
	var prefix sql.NullString
	if req.RepositoryID != nil {
		s.db.WithContext(ctx).Table("repositories").
			Where("id = ?", *req.RepositoryID).
			Select("ticket_prefix").
			Scan(&prefix)
	}

	if prefix.Valid && prefix.String != "" {
		// Repository has a prefix: generate number scoped to repository
		var maxNumber int
		s.db.WithContext(ctx).Model(&ticket.Ticket{}).
			Where("repository_id = ?", req.RepositoryID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber)
		number = maxNumber + 1
		identifier = fmt.Sprintf("%s-%d", prefix.String, number)
	} else {
		// No prefix: generate number scoped to organization with TICKET- prefix
		var maxNumber int
		s.db.WithContext(ctx).Model(&ticket.Ticket{}).
			Where("organization_id = ? AND identifier LIKE 'TICKET-%'", req.OrganizationID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNumber)
		number = maxNumber + 1
		identifier = fmt.Sprintf("TICKET-%d", number)
	}

	status := req.Status
	if status == "" {
		status = ticket.TicketStatusBacklog
	}

	t := &ticket.Ticket{
		OrganizationID: req.OrganizationID,
		Number:         number,
		Identifier:     identifier,
		Type:           req.Type,
		Title:          req.Title,
		Description:    req.Description,
		Content:        req.Content,
		Status:         status,
		Priority:       req.Priority,
		DueDate:        req.DueDate,
		RepositoryID:   req.RepositoryID,
		ReporterID:     req.ReporterID,
		ParentTicketID: req.ParentTicketID,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(t).Error; err != nil {
			return err
		}

		// Add assignees
		for _, userID := range req.AssigneeIDs {
			assignee := &ticket.Assignee{
				TicketID: t.ID,
				UserID:   userID,
			}
			if err := tx.Create(assignee).Error; err != nil {
				return err
			}
		}

		// Add labels by ID
		for _, labelID := range req.LabelIDs {
			ticketLabel := &ticket.TicketLabel{
				TicketID: t.ID,
				LabelID:  labelID,
			}
			if err := tx.Create(ticketLabel).Error; err != nil {
				return err
			}
		}

		// Add labels by name (if provided)
		for _, labelName := range req.Labels {
			var label ticket.Label
			if err := tx.Where("organization_id = ? AND name = ?", req.OrganizationID, labelName).First(&label).Error; err != nil {
				continue // Skip if label not found
			}
			ticketLabel := &ticket.TicketLabel{
				TicketID: t.ID,
				LabelID:  label.ID,
			}
			if err := tx.Create(ticketLabel).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get the created ticket with full details
	createdTicket, err := s.GetTicket(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	// Publish ticket created event (Service layer - Information Expert)
	s.publishEvent(ctx, TicketEventCreated, req.OrganizationID, createdTicket.Identifier, createdTicket.Status, "")

	return createdTicket, nil
}

// GetTicket returns a ticket by ID
func (s *Service) GetTicket(ctx context.Context, ticketID int64) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Assignees").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		First(&t, ticketID).Error; err != nil {
		return nil, ErrTicketNotFound
	}
	return &t, nil
}

// GetTicketByIdentifier returns a ticket by identifier
func (s *Service) GetTicketByIdentifier(ctx context.Context, identifier string) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := s.db.WithContext(ctx).
		Preload("Assignees").
		Preload("Labels").
		Preload("MergeRequests").
		Preload("SubTickets").
		Where("identifier = ?", identifier).
		First(&t).Error; err != nil {
		return nil, ErrTicketNotFound
	}
	return &t, nil
}

// ListTickets returns tickets based on filters
func (s *Service) ListTickets(ctx context.Context, filter *ListTicketsFilter) ([]*ticket.Ticket, int64, error) {
	query := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("organization_id = ?", filter.OrganizationID)

	if filter.RepositoryID != nil {
		query = query.Where("repository_id = ?", *filter.RepositoryID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Priority != "" {
		query = query.Where("priority = ?", filter.Priority)
	}
	if filter.ReporterID != nil {
		query = query.Where("reporter_id = ?", *filter.ReporterID)
	}
	if filter.ParentTicketID != nil {
		query = query.Where("parent_ticket_id = ?", *filter.ParentTicketID)
	}
	if filter.Query != "" {
		query = query.Where("title ILIKE ? OR identifier ILIKE ?", "%"+filter.Query+"%", "%"+filter.Query+"%")
	}
	if filter.AssigneeID != nil {
		query = query.Joins("JOIN ticket_assignees ON ticket_assignees.ticket_id = tickets.id").
			Where("ticket_assignees.user_id = ?", *filter.AssigneeID)
	}
	if len(filter.LabelIDs) > 0 {
		query = query.Joins("JOIN ticket_labels ON ticket_labels.ticket_id = tickets.id").
			Where("ticket_labels.label_id IN ?", filter.LabelIDs)
	}

	var total int64
	query.Count(&total)

	var tickets []*ticket.Ticket
	if err := query.
		Preload("Assignees").
		Preload("Labels").
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// UpdateTicket updates a ticket
func (s *Service) UpdateTicket(ctx context.Context, ticketID int64, updates map[string]interface{}) (*ticket.Ticket, error) {
	// Get the ticket before update to capture previous status
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	previousStatus := oldTicket.Status

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	updatedTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	// Publish appropriate event based on what changed
	if newStatus, ok := updates["status"].(string); ok && newStatus != previousStatus {
		s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, updatedTicket.Identifier, updatedTicket.Status, previousStatus)
	} else {
		s.publishEvent(ctx, TicketEventUpdated, oldTicket.OrganizationID, updatedTicket.Identifier, updatedTicket.Status, previousStatus)
	}

	return updatedTicket, nil
}

// UpdateStatus updates a ticket's status
func (s *Service) UpdateStatus(ctx context.Context, ticketID int64, status string) error {
	// Get the ticket before update to capture previous status and org ID
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	previousStatus := oldTicket.Status

	updates := map[string]interface{}{
		"status": status,
	}

	now := time.Now()
	switch status {
	case ticket.TicketStatusInProgress:
		updates["started_at"] = now
	case ticket.TicketStatusDone:
		updates["completed_at"] = now
	}

	if err := s.db.WithContext(ctx).Model(&ticket.Ticket{}).Where("id = ?", ticketID).Updates(updates).Error; err != nil {
		return err
	}

	// Publish status changed event (for kanban board real-time updates)
	s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, oldTicket.Identifier, status, previousStatus)

	return nil
}

// DeleteTicket deletes a ticket
func (s *Service) DeleteTicket(ctx context.Context, ticketID int64) error {
	// Get the ticket before deletion to capture info for event
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Delete(&ticket.Ticket{}, ticketID).Error; err != nil {
		return err
	}

	// Publish ticket deleted event
	s.publishEvent(ctx, TicketEventDeleted, oldTicket.OrganizationID, oldTicket.Identifier, "deleted", oldTicket.Status)

	return nil
}
