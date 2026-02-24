package ticket

import "time"

// Comment represents a comment on a ticket
type Comment struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	TicketID int64  `gorm:"not null;index" json:"ticket_id"`
	UserID   int64  `gorm:"not null;index" json:"user_id"`
	Content  string `gorm:"type:text;not null" json:"content"`
	ParentID *int64 `gorm:"index" json:"parent_id,omitempty"`

	Mentions []CommentMention `gorm:"type:jsonb;serializer:json" json:"mentions,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	User    *AssigneeUser `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []Comment     `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

func (Comment) TableName() string {
	return "ticket_comments"
}

// CommentMention represents a user mention within a comment
type CommentMention struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}
