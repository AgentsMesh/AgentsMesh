package channel

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

var (
	ErrChannelNotFound  = errors.New("channel not found")
	ErrChannelArchived  = errors.New("channel is archived")
	ErrDuplicateName    = errors.New("channel name already exists")
	ErrMessageNotFound  = errors.New("message not found")
	ErrNotMessageSender = errors.New("only the message sender can perform this action")
)

// Service handles channel operations
type Service struct {
	repo          channel.ChannelRepository
	eventBus      *eventbus.EventBus
	postSendHooks []PostSendHook
}

// NewService creates a new channel service
func NewService(repo channel.ChannelRepository) *Service {
	return &Service{repo: repo}
}

// SetEventBus sets the event bus for publishing channel events
func (s *Service) SetEventBus(eb *eventbus.EventBus) {
	s.eventBus = eb
}

// AddPostSendHook registers a hook to be called after message persistence
func (s *Service) AddPostSendHook(hook PostSendHook) {
	s.postSendHooks = append(s.postSendHooks, hook)
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
		slog.Error("failed to create channel", "org_id", req.OrganizationID, "name", req.Name, "error", err)
		return nil, err
	}

	slog.Info("channel created", "channel_id", ch.ID, "org_id", req.OrganizationID, "name", req.Name)
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

// ListChannels returns channels for an organization with optional filters.
func (s *Service) ListChannels(ctx context.Context, orgID int64, filter *channel.ChannelListFilter) ([]*channel.Channel, int64, error) {
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
			slog.Error("failed to update channel", "channel_id", channelID, "error", err)
			return nil, err
		}
		slog.Info("channel updated", "channel_id", channelID)
	}

	return s.GetChannel(ctx, channelID)
}

// ArchiveChannel archives a channel
func (s *Service) ArchiveChannel(ctx context.Context, channelID int64) error {
	if err := s.repo.SetArchived(ctx, channelID, true); err != nil {
		slog.Error("failed to archive channel", "channel_id", channelID, "error", err)
		return err
	}
	slog.Info("channel archived", "channel_id", channelID)
	return nil
}

// UnarchiveChannel unarchives a channel
func (s *Service) UnarchiveChannel(ctx context.Context, channelID int64) error {
	if err := s.repo.SetArchived(ctx, channelID, false); err != nil {
		slog.Error("failed to unarchive channel", "channel_id", channelID, "error", err)
		return err
	}
	slog.Info("channel unarchived", "channel_id", channelID)
	return nil
}

// DeleteChannel permanently deletes a channel and all associated data.
// Prefer ArchiveChannel for soft-delete; use this only for hard-delete scenarios.
func (s *Service) DeleteChannel(ctx context.Context, channelID int64) error {
	if err := s.repo.DeleteWithCleanup(ctx, channelID); err != nil {
		slog.Error("failed to delete channel", "channel_id", channelID, "error", err)
		return err
	}
	slog.Info("channel deleted", "channel_id", channelID)
	return nil
}

// DeleteChannelsByOrg deletes all channels for an organization (used during org deletion)
func (s *Service) DeleteChannelsByOrg(ctx context.Context, orgID int64) error {
	if err := s.repo.DeleteChannelsByOrg(ctx, orgID); err != nil {
		slog.Error("failed to delete channels by org", "org_id", orgID, "error", err)
		return err
	}
	slog.Info("channels deleted for org", "org_id", orgID)
	return nil
}

// CleanupUserReferences removes a user's channel membership/read-state data
func (s *Service) CleanupUserReferences(ctx context.Context, userID int64) error {
	return s.repo.CleanupUserReferences(ctx, userID)
}

// GetChannelsByTicket returns channels for a ticket
func (s *Service) GetChannelsByTicket(ctx context.Context, ticketID int64) ([]*channel.Channel, error) {
	return s.repo.GetByTicketID(ctx, ticketID)
}
