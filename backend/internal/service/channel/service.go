package channel

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"gorm.io/gorm"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrChannelArchived = errors.New("channel is archived")
	ErrDuplicateName   = errors.New("channel name already exists")
)

// Service handles channel operations
type Service struct {
	db       *gorm.DB
	eventBus *eventbus.EventBus
}

// NewService creates a new channel service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// SetEventBus sets the event bus for publishing channel events
func (s *Service) SetEventBus(eb *eventbus.EventBus) {
	s.eventBus = eb
}

// CreateChannelRequest represents a channel creation request
type CreateChannelRequest struct {
	OrganizationID  int64
	Name            string
	Description     *string
	RepositoryID    *int64
	TicketID        *int64
	CreatedByPod    *string
	CreatedByUserID *int64
}

// CreateChannel creates a new channel
func (s *Service) CreateChannel(ctx context.Context, req *CreateChannelRequest) (*channel.Channel, error) {
	var existing channel.Channel
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name = ?", req.OrganizationID, req.Name).
		First(&existing).Error; err == nil {
		return nil, ErrDuplicateName
	}

	ch := &channel.Channel{
		OrganizationID:  req.OrganizationID,
		Name:            req.Name,
		Description:     req.Description,
		RepositoryID:    req.RepositoryID,
		TicketID:        req.TicketID,
		CreatedByPod:    req.CreatedByPod,
		CreatedByUserID: req.CreatedByUserID,
		IsArchived:      false,
	}

	if err := s.db.WithContext(ctx).Create(ch).Error; err != nil {
		return nil, err
	}

	return ch, nil
}

// GetChannel returns a channel by ID
func (s *Service) GetChannel(ctx context.Context, channelID int64) (*channel.Channel, error) {
	var ch channel.Channel
	if err := s.db.WithContext(ctx).First(&ch, channelID).Error; err != nil {
		return nil, ErrChannelNotFound
	}
	return &ch, nil
}

// GetChannelByName returns a channel by name within an organization
func (s *Service) GetChannelByName(ctx context.Context, orgID int64, name string) (*channel.Channel, error) {
	var ch channel.Channel
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name = ?", orgID, name).
		First(&ch).Error; err != nil {
		return nil, ErrChannelNotFound
	}
	return &ch, nil
}

// ListChannels returns channels for an organization
func (s *Service) ListChannels(ctx context.Context, orgID int64, includeArchived bool, limit, offset int) ([]*channel.Channel, int64, error) {
	query := s.db.WithContext(ctx).Model(&channel.Channel{}).Where("organization_id = ?", orgID)

	if !includeArchived {
		query = query.Where("is_archived = ?", false)
	}

	var total int64
	query.Count(&total)

	var channels []*channel.Channel
	if err := query.
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// UpdateChannel updates a channel
func (s *Service) UpdateChannel(ctx context.Context, channelID int64, name, description, document *string) (*channel.Channel, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	updates := make(map[string]interface{})
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if document != nil {
		updates["document"] = *document
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(ch).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetChannel(ctx, channelID)
}

// ArchiveChannel archives a channel
func (s *Service) ArchiveChannel(ctx context.Context, channelID int64) error {
	return s.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("is_archived", true).Error
}

// UnarchiveChannel unarchives a channel
func (s *Service) UnarchiveChannel(ctx context.Context, channelID int64) error {
	return s.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("is_archived", false).Error
}

// GetChannelsByTicket returns channels for a ticket
func (s *Service) GetChannelsByTicket(ctx context.Context, ticketID int64) ([]*channel.Channel, error) {
	var channels []*channel.Channel
	if err := s.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}
