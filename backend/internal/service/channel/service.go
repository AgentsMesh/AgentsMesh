package channel

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrChannelArchived = errors.New("channel is archived")
	ErrDuplicateName   = errors.New("channel name already exists")
)

// Service handles channel operations
type Service struct {
	repo     channel.ChannelRepository
	eventBus *eventbus.EventBus
}

// NewService creates a new channel service
func NewService(repo channel.ChannelRepository) *Service {
	return &Service{repo: repo}
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
	existing, err := s.repo.GetByOrgAndName(ctx, req.OrganizationID, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
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

	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// GetChannel returns a channel by ID
func (s *Service) GetChannel(ctx context.Context, channelID int64) (*channel.Channel, error) {
	ch, err := s.repo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

// GetChannelByName returns a channel by name within an organization
func (s *Service) GetChannelByName(ctx context.Context, orgID int64, name string) (*channel.Channel, error) {
	ch, err := s.repo.GetByOrgAndName(ctx, orgID, name)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, ErrChannelNotFound
	}
	return ch, nil
}

// ListChannelsFilter contains optional filters for listing channels.
// Kept for backward compatibility; delegates to domain type.
type ListChannelsFilter = channel.ChannelListFilter

// ListChannels returns channels for an organization with optional filters.
func (s *Service) ListChannels(ctx context.Context, orgID int64, filter *ListChannelsFilter) ([]*channel.Channel, int64, error) {
	return s.repo.ListByOrg(ctx, orgID, filter)
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
		if err := s.repo.UpdateFields(ctx, channelID, updates); err != nil {
			return nil, err
		}
	}

	return s.GetChannel(ctx, channelID)
}

// ArchiveChannel archives a channel
func (s *Service) ArchiveChannel(ctx context.Context, channelID int64) error {
	return s.repo.SetArchived(ctx, channelID, true)
}

// UnarchiveChannel unarchives a channel
func (s *Service) UnarchiveChannel(ctx context.Context, channelID int64) error {
	return s.repo.SetArchived(ctx, channelID, false)
}

// GetChannelsByTicket returns channels for a ticket
func (s *Service) GetChannelsByTicket(ctx context.Context, ticketID int64) ([]*channel.Channel, error) {
	return s.repo.GetByTicketID(ctx, ticketID)
}
