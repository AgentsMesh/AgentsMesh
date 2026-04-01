package ticket

import (
	"time"
)

// Relation type constants
const (
	RelationTypeBlocks    = "blocks"
	RelationTypeBlockedBy = "blocked_by"
	RelationTypeRelates   = "relates_to"
	RelationTypeDuplicate = "duplicates"
)

// Relation represents a relationship between two tickets
type Relation struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	SourceTicketID int64  `gorm:"not null;index" json:"source_ticket_id"`
	TargetTicketID int64  `gorm:"not null;index" json:"target_ticket_id"`
	RelationType   string `gorm:"size:50;not null" json:"relation_type"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Associations
	SourceTicket *Ticket `gorm:"foreignKey:SourceTicketID" json:"source_ticket,omitempty"`
	TargetTicket *Ticket `gorm:"foreignKey:TargetTicketID" json:"target_ticket,omitempty"`
}

func (Relation) TableName() string {
	return "ticket_relations"
}
