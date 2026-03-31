package channel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/lib/pq"
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

	CreatedByPod *string `gorm:"size:100" json:"created_by_pod,omitempty"`
	CreatedByUserID  *int64  `json:"created_by_user_id,omitempty"`

	IsArchived bool `gorm:"not null;default:false" json:"is_archived"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Messages []Message `gorm:"foreignKey:ChannelID" json:"messages,omitempty"`
}

func (Channel) TableName() string {
	return "channels"
}

// Message type constants
const (
	MessageTypeText    = "text"
	MessageTypeSystem  = "system"
	MessageTypeCode    = "code"
	MessageTypeCommand = "command"
)

// MessageMetadata represents optional message metadata
type MessageMetadata map[string]interface{}

// Scan implements sql.Scanner for MessageMetadata
func (mm *MessageMetadata) Scan(value interface{}) error {
	if value == nil {
		*mm = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for Scan")
	}
	return json.Unmarshal(bytes, mm)
}

// Value implements driver.Valuer for MessageMetadata
func (mm MessageMetadata) Value() (driver.Value, error) {
	if mm == nil {
		return nil, nil
	}
	return json.Marshal(mm)
}

// Message represents a message in a channel
type Message struct {
	ID        int64 `gorm:"primaryKey" json:"id"`
	ChannelID int64 `gorm:"not null;index" json:"channel_id"`

	SenderPod *string `gorm:"size:100" json:"sender_pod,omitempty"`
	SenderUserID  *int64  `json:"sender_user_id,omitempty"`

	MessageType string          `gorm:"size:50;not null;default:'text'" json:"message_type"`
	Content     string          `gorm:"type:text;not null" json:"content"`
	Metadata    MessageMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`

	EditedAt  *time.Time `gorm:"column:edited_at" json:"edited_at,omitempty"`
	IsDeleted bool       `gorm:"default:false" json:"is_deleted,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`

	// Associations
	Channel       *Channel       `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	SenderUser    *user.User     `gorm:"foreignKey:SenderUserID" json:"sender_user,omitempty"`
	SenderPodInfo *agentpod.Pod  `gorm:"foreignKey:SenderPod;references:PodKey" json:"sender_pod_info,omitempty"`
}

func (Message) TableName() string {
	return "channel_messages"
}

// Binding status constants
const (
	BindingStatusPending  = "pending"
	BindingStatusActive   = "active"
	BindingStatusRejected = "rejected"
	BindingStatusInactive = "inactive"
	BindingStatusExpired  = "expired"
)

// Binding scope constants
const (
	BindingScopePodRead  = "pod:read"
	BindingScopePodWrite = "pod:write"
)

// Binding policy constants
const (
	BindingPolicySameUserAuto    = "same_user_auto"
	BindingPolicySameProjectAuto = "same_project_auto"
	BindingPolicyExplicitOnly    = "explicit_only"
)

// ValidBindingScopes contains all valid binding scopes
var ValidBindingScopes = map[string]bool{
	BindingScopePodRead:  true,
	BindingScopePodWrite: true,
}

// PodBinding represents a binding between two pods
type PodBinding struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	InitiatorPod  string         `gorm:"size:100;not null;index" json:"initiator_pod"`
	TargetPod     string         `gorm:"size:100;not null;index" json:"target_pod"`
	GrantedScopes pq.StringArray `gorm:"type:text[]" json:"granted_scopes"`
	PendingScopes pq.StringArray `gorm:"type:text[]" json:"pending_scopes"`
	Status        string         `gorm:"size:50;not null;default:'pending'" json:"status"`

	// Timestamps
	RequestedAt *time.Time `json:"requested_at,omitempty"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Rejection information
	RejectionReason *string `gorm:"size:500" json:"rejection_reason,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (PodBinding) TableName() string {
	return "pod_bindings"
}

// HasScope checks if a specific scope is granted
func (b *PodBinding) HasScope(scope string) bool {
	for _, s := range b.GrantedScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasPendingScope checks if a specific scope is pending approval
func (b *PodBinding) HasPendingScope(scope string) bool {
	for _, s := range b.PendingScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsActive checks if the binding is currently active
func (b *PodBinding) IsActive() bool {
	return b.Status == BindingStatusActive
}

// IsPending checks if the binding is pending approval
func (b *PodBinding) IsPending() bool {
	return b.Status == BindingStatusPending
}

// CanObserve checks if the initiator can observe the target's pod
func (b *PodBinding) CanObserve() bool {
	return b.IsActive() && b.HasScope(BindingScopePodRead)
}

// CanControl checks if the initiator can send input to the target's pod
func (b *PodBinding) CanControl() bool {
	return b.IsActive() && b.HasScope(BindingScopePodWrite)
}
