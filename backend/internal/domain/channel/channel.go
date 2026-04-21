package channel

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

const (
	RoleCreator = "creator"
	RoleMember  = "member"
)

// Channel represents a communication channel for agent collaboration
type Channel struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	Name        string  `gorm:"size:100;not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	Document    *string `gorm:"type:text" json:"document,omitempty"` // Shared document

	RepositoryID *int64 `json:"repository_id,omitempty"`
	TicketID     *int64 `json:"ticket_id,omitempty"`

	CreatedByPod    *string `gorm:"size:100" json:"created_by_pod,omitempty"`
	CreatedByUserID *int64  `json:"created_by_user_id,omitempty"`

	Visibility string `gorm:"size:10;not null;default:'public'" json:"visibility"`
	IsArchived bool   `gorm:"not null;default:false" json:"is_archived"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Computed fields (populated by query, not persisted)
	IsMember    bool  `gorm:"-" json:"is_member"`
	MemberCount int64 `gorm:"-" json:"member_count"`

	// Associations
	Messages []Message `gorm:"foreignKey:ChannelID" json:"messages,omitempty"`
}

func (c *Channel) IsPublic() bool {
	return c.Visibility == "" || c.Visibility == VisibilityPublic
}

func (Channel) TableName() string {
	return "channels"
}

// Message type constants
const (
	MessageTypeText       = "text"
	MessageTypeAttachment = "attachment"
	MessageTypeSystem     = "system"
)

// Message represents a message in a channel
type Message struct {
	ID        int64 `gorm:"primaryKey" json:"id"`
	ChannelID int64 `gorm:"not null;index" json:"channel_id"`

	SenderPod    *string `gorm:"size:100" json:"sender_pod,omitempty"`
	SenderUserID *int64  `json:"sender_user_id,omitempty"`

	MessageType string          `gorm:"size:50;not null;default:'text'" json:"message_type"`
	Body        string          `gorm:"type:text;not null" json:"body"`
	Content     *MessageContent `gorm:"type:jsonb" json:"content,omitempty"`
	Mentions    MessageMentions `gorm:"type:jsonb;default:'{}'" json:"mentions"`
	ReplyTo     *int64          `json:"reply_to,omitempty"`

	EditedAt  *time.Time `gorm:"column:edited_at" json:"edited_at,omitempty"`
	IsDeleted bool       `gorm:"default:false" json:"is_deleted,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`

	// Associations
	Channel       *Channel      `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	SenderUser    *user.User    `gorm:"foreignKey:SenderUserID" json:"sender_user,omitempty"`
	SenderPodInfo *agentpod.Pod `gorm:"foreignKey:SenderPod;references:PodKey" json:"sender_pod_info,omitempty"`
}

func (Message) TableName() string {
	return "channel_messages"
}
