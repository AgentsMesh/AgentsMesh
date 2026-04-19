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
	ErrEmptyContent     = errors.New("message content cannot be empty")
	ErrInvalidContent   = errors.New("invalid message content")
)

// Service handles channel operations
type Service struct {
	repo               channel.ChannelRepository
	eventBus           *eventbus.EventBus
	postSendHooks      []PostSendHook
	podCreatorResolver PodCreatorResolver
	userLookup         UserLookup
}

func NewService(repo channel.ChannelRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetEventBus(eb *eventbus.EventBus) {
	s.eventBus = eb
}

func (s *Service) SetUserLookup(lookup UserLookup) {
	s.userLookup = lookup
}

// AddPostSendHook registers a hook to be called after message persistence
func (s *Service) AddPostSendHook(hook PostSendHook) {
	s.postSendHooks = append(s.postSendHooks, hook)
}

// CreateChannelRequest represents a channel creation request
type CreateChannelRequest struct {
	OrganizationID   int64
	Name             string
	Description      *string
	RepositoryID     *int64
	TicketID         *int64
	CreatedByPod     *string
	CreatedByUserID  *int64
	Visibility       string
	InitialMemberIDs []int64
}

// CreateChannel creates a new channel with explicit membership
func (s *Service) CreateChannel(ctx context.Context, req *CreateChannelRequest) (*channel.Channel, error) {
	existing, err := s.repo.GetByOrgAndName(ctx, req.OrganizationID, req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDuplicateName
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = channel.VisibilityPublic
	}

	ch := &channel.Channel{
		OrganizationID:  req.OrganizationID,
		Name:            req.Name,
		Description:     req.Description,
		RepositoryID:    req.RepositoryID,
		TicketID:        req.TicketID,
		CreatedByPod:    req.CreatedByPod,
		CreatedByUserID: req.CreatedByUserID,
		Visibility:      visibility,
		IsArchived:      false,
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		slog.ErrorContext(ctx, "failed to create channel", "org_id", req.OrganizationID, "name", req.Name, "error", err)
		return nil, err
	}

	// Creator auto-joins with creator role
	if req.CreatedByUserID != nil {
		_ = s.repo.AddMemberWithRole(ctx, ch.ID, *req.CreatedByUserID, channel.RoleCreator)
	}
	validMembers := s.validateOrgMembers(ctx, req.OrganizationID, req.InitialMemberIDs)
	for _, uid := range validMembers {
		if req.CreatedByUserID != nil && uid == *req.CreatedByUserID {
			continue
		}
		_ = s.repo.AddMemberWithRole(ctx, ch.ID, uid, channel.RoleMember)
	}

	slog.InfoContext(ctx, "channel created", "channel_id", ch.ID, "org_id", req.OrganizationID, "name", req.Name, "visibility", visibility)
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

// GetChannelForUser returns a channel by ID with membership info populated.
func (s *Service) GetChannelForUser(ctx context.Context, channelID, userID int64) (*channel.Channel, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	ch.IsMember, _ = s.repo.IsMember(ctx, channelID, userID)
	memberIDs, _ := s.repo.GetMemberUserIDs(ctx, channelID)
	ch.MemberCount = int64(len(memberIDs))
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

// ListChannels returns channels visible to a user within an organization.
// Public channels are always visible; private channels require membership.
func (s *Service) ListChannels(ctx context.Context, orgID, userID int64, filter *channel.ChannelListFilter) ([]*channel.Channel, int64, error) {
	return s.repo.ListVisibleForUser(ctx, orgID, userID, filter)
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
			slog.ErrorContext(ctx, "failed to update channel", "channel_id", channelID, "error", err)
			return nil, err
		}
		slog.InfoContext(ctx, "channel updated", "channel_id", channelID)
	}

	return s.GetChannel(ctx, channelID)
}
