package supportticket

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

const (
	CategoryBug            = "bug"
	CategoryFeatureRequest = "feature_request"
	CategoryUsageQuestion  = "usage_question"
	CategoryAccount        = "account"
	CategoryOther          = "other"
)

const (
	StatusOpen       = "open"
	StatusInProgress = "in_progress"
	StatusResolved   = "resolved"
	StatusClosed     = "closed"
)

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

var ValidCategories = map[string]bool{
	CategoryBug:            true,
	CategoryFeatureRequest: true,
	CategoryUsageQuestion:  true,
	CategoryAccount:        true,
	CategoryOther:          true,
}

var ValidStatuses = map[string]bool{
	StatusOpen:       true,
	StatusInProgress: true,
	StatusResolved:   true,
	StatusClosed:     true,
}

var ValidTransitions = map[string]map[string]bool{
	StatusOpen:       {StatusInProgress: true, StatusResolved: true, StatusClosed: true},
	StatusInProgress: {StatusOpen: true, StatusResolved: true, StatusClosed: true},
	StatusResolved:   {StatusOpen: true, StatusClosed: true},
	StatusClosed:     {StatusOpen: true},
}

var ValidPriorities = map[string]bool{
	PriorityLow:    true,
	PriorityMedium: true,
	PriorityHigh:   true,
}

type SupportTicket struct {
	ID              int64      `gorm:"primaryKey" json:"id"`
	UserID          int64      `gorm:"not null;index" json:"user_id"`
	Title           string     `gorm:"size:255;not null" json:"title"`
	Category        string     `gorm:"size:50;not null;default:other" json:"category"`
	Status          string     `gorm:"size:50;not null;default:open" json:"status"`
	Priority        string     `gorm:"size:20;not null;default:medium" json:"priority"`
	AssignedAdminID *int64     `gorm:"index" json:"assigned_admin_id,omitempty"`
	CreatedAt       time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`

	User          *user.User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	AssignedAdmin *user.User             `gorm:"foreignKey:AssignedAdminID" json:"assigned_admin,omitempty"`
	Messages      []SupportTicketMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
}

func (SupportTicket) TableName() string {
	return "support_tickets"
}

type SupportTicketMessage struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	TicketID     int64     `gorm:"not null;index" json:"ticket_id"`
	UserID       int64     `gorm:"not null" json:"user_id"`
	Content      string    `gorm:"type:text;not null" json:"content"`
	IsAdminReply bool      `gorm:"not null;default:false" json:"is_admin_reply"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`

	User        *user.User                  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Attachments []SupportTicketAttachment    `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}

func (SupportTicketMessage) TableName() string {
	return "support_ticket_messages"
}

type SupportTicketAttachment struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	TicketID     int64     `gorm:"not null;index" json:"ticket_id"`
	MessageID    *int64    `gorm:"index" json:"message_id,omitempty"`
	UploaderID   int64     `gorm:"not null" json:"uploader_id"`
	OriginalName string    `gorm:"size:255;not null" json:"original_name"`
	StorageKey   string    `gorm:"size:500;not null" json:"-"`
	MimeType     string    `gorm:"size:100;not null" json:"mime_type"`
	Size         int64     `gorm:"not null" json:"size"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (SupportTicketAttachment) TableName() string {
	return "support_ticket_attachments"
}
