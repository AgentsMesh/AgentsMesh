package channel

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

type ChannelListFilter struct {
	IncludeArchived bool
	RepositoryID    *int64
	TicketID        *int64
	Visibility      *string
	Limit           int
	Offset          int
}

type ChannelStore interface {
	GetByID(ctx context.Context, channelID int64) (*Channel, error)
	GetByOrgAndName(ctx context.Context, orgID int64, name string) (*Channel, error)
	GetByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*Channel, error)
	SlugExists(ctx context.Context, orgID int64, slug string) (bool, error)
	Create(ctx context.Context, ch *Channel) error
	ListByOrg(ctx context.Context, orgID int64, filter *ChannelListFilter) ([]*Channel, int64, error)
	ListVisibleForUser(ctx context.Context, orgID, userID int64, filter *ChannelListFilter) ([]*Channel, int64, error)
	UpdateFields(ctx context.Context, channelID int64, updates map[string]interface{}) error
	SetArchived(ctx context.Context, channelID int64, archived bool) error
	GetByTicketID(ctx context.Context, ticketID int64) ([]*Channel, error)
	TouchChannel(ctx context.Context, channelID int64) error
}

type MessageStore interface {
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, channelID int64, before *time.Time, after *time.Time, limit int) ([]*Message, error)
	GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*Message, bool, error)
	GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*Message, error)
	GetMessagesBefore(ctx context.Context, channelID int64, beforeID int64, limit int) ([]*Message, error)
	GetMessageByID(ctx context.Context, messageID int64) (*Message, error)
	UpdateMessage(ctx context.Context, messageID int64, body string, content *MessageContent, mentions MessageMentions) error
	UpdateMessageMentions(ctx context.Context, messageID int64, mentions MessageMentions) error
	SoftDeleteMessage(ctx context.Context, messageID int64) error
	SearchMessages(ctx context.Context, channelID int64, query string, limit int) ([]*Message, error)
	SaveMessageEdit(ctx context.Context, edit *MessageEdit) error
	GetMessageEdits(ctx context.Context, messageID int64) ([]*MessageEdit, error)
}

type MemberStore interface {
	UpsertMember(ctx context.Context, channelID, userID int64) error
	AddMemberWithRole(ctx context.Context, channelID, userID int64, role string) error
	IsMember(ctx context.Context, channelID, userID int64) (bool, error)
	GetMemberRole(ctx context.Context, channelID, userID int64) (string, error)
	RemoveMember(ctx context.Context, channelID, userID int64) error
	GetMembers(ctx context.Context, channelID int64, limit, offset int) ([]Member, int64, error)
	GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
	GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
	SetMemberMuted(ctx context.Context, channelID, userID int64, muted bool) error
	MarkRead(ctx context.Context, channelID, userID int64, messageID int64) error
	GetUnreadCounts(ctx context.Context, userID int64) (map[int64]int64, error)
}

type AccessStore interface {
	UpsertAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error
	GetChannelsForPod(ctx context.Context, podKey string) ([]*Channel, error)
	HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error)
	GetAccessCount(ctx context.Context, channelID int64) (int64, error)
	AddPodToChannel(ctx context.Context, channelID int64, podKey string) error
	RemovePodFromChannel(ctx context.Context, channelID int64, podKey string) error
	GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error)
	GetChannelPodCount(ctx context.Context, channelID int64) (int64, error)
	CreateBinding(ctx context.Context, binding *PodBinding) error
	GetBindingByID(ctx context.Context, bindingID int64) (*PodBinding, error)
	GetBindingByPods(ctx context.Context, initiator, target string) (*PodBinding, error)
	ListBindingsForPod(ctx context.Context, podKey string) ([]*PodBinding, error)
	UpdateBindingFields(ctx context.Context, bindingID int64, updates map[string]interface{}) error
}

type CleanupStore interface {
	DeleteWithCleanup(ctx context.Context, channelID int64) error
	DeleteChannelsByOrg(ctx context.Context, orgID int64) error
	CleanupUserReferences(ctx context.Context, userID int64) error
}

// ChannelRepository is the composite interface for backward compatibility.
// Service layer depends on this; future services can depend on narrower sub-interfaces.
type ChannelRepository interface {
	ChannelStore
	MessageStore
	MemberStore
	AccessStore
	CleanupStore
}
