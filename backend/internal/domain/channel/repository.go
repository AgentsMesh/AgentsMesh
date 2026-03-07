package channel

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// ChannelListFilter contains optional filters for listing channels.
type ChannelListFilter struct {
	IncludeArchived bool
	RepositoryID    *int64
	TicketID        *int64
	Limit           int
	Offset          int
}

// ChannelRepository defines the data access interface for channel operations.
type ChannelRepository interface {
	// --- Channel CRUD ---

	// GetByID returns a channel by primary key. Returns (nil, nil) if not found.
	GetByID(ctx context.Context, channelID int64) (*Channel, error)

	// GetByOrgAndName returns a channel by org and name. Returns (nil, nil) if not found.
	GetByOrgAndName(ctx context.Context, orgID int64, name string) (*Channel, error)

	// Create inserts a new channel record.
	Create(ctx context.Context, ch *Channel) error

	// ListByOrg returns channels for an organization with pagination and count.
	ListByOrg(ctx context.Context, orgID int64, filter *ChannelListFilter) ([]*Channel, int64, error)

	// UpdateFields applies partial updates to a channel by ID.
	UpdateFields(ctx context.Context, channelID int64, updates map[string]interface{}) error

	// SetArchived sets the archived flag on a channel.
	SetArchived(ctx context.Context, channelID int64, archived bool) error

	// GetByTicketID returns channels associated with a ticket.
	GetByTicketID(ctx context.Context, ticketID int64) ([]*Channel, error)

	// --- Messages ---

	// CreateMessage inserts a new message record.
	CreateMessage(ctx context.Context, msg *Message) error

	// TouchChannel updates the channel's updated_at timestamp.
	TouchChannel(ctx context.Context, channelID int64) error

	// GetMessages returns messages with sender preloads, ordered by created_at DESC, limited.
	GetMessages(ctx context.Context, channelID int64, before *time.Time, limit int) ([]*Message, error)

	// GetMessagesMentioning returns messages containing a pattern, ordered DESC, limited.
	GetMessagesMentioning(ctx context.Context, channelID int64, pattern string, limit int) ([]*Message, error)

	// GetRecentMessages returns the most recent messages, ordered DESC, limited.
	GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*Message, error)

	// --- Access tracking ---

	// UpsertAccess creates or updates an access record for a pod or user on a channel.
	UpsertAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error

	// GetChannelsForPod returns channels a pod has accessed.
	GetChannelsForPod(ctx context.Context, podKey string) ([]*Channel, error)

	// HasAccessed checks if a pod has accessed a channel.
	HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error)

	// GetAccessCount returns the number of unique accessors for a channel.
	GetAccessCount(ctx context.Context, channelID int64) (int64, error)

	// --- Channel Pods ---

	// AddPodToChannel associates a pod with a channel.
	AddPodToChannel(ctx context.Context, channelID int64, podKey string) error

	// RemovePodFromChannel disassociates a pod from a channel.
	RemovePodFromChannel(ctx context.Context, channelID int64, podKey string) error

	// GetChannelPods returns Pod entities joined to a channel.
	GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error)

	// --- Bindings (channel-level) ---

	// CreateBinding inserts a new binding record.
	CreateBinding(ctx context.Context, binding *PodBinding) error

	// GetBindingByID returns a binding by primary key. Returns (nil, nil) if not found.
	GetBindingByID(ctx context.Context, bindingID int64) (*PodBinding, error)

	// GetBindingByPods returns a binding by initiator and target. Returns (nil, nil) if not found.
	GetBindingByPods(ctx context.Context, initiator, target string) (*PodBinding, error)

	// ListBindingsForPod returns all bindings where the pod is initiator or target.
	ListBindingsForPod(ctx context.Context, podKey string) ([]*PodBinding, error)

	// UpdateBindingFields applies partial updates to a binding by ID.
	UpdateBindingFields(ctx context.Context, bindingID int64, updates map[string]interface{}) error
}
