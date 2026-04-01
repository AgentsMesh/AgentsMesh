package ticket

import (
	"time"
)

// Commit represents a git commit linked to a ticket
type Commit struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	TicketID     int64  `gorm:"not null;index" json:"ticket_id"`
	RepositoryID int64  `gorm:"not null;index" json:"repository_id"`
	PodID        *int64 `json:"pod_id,omitempty"`

	CommitSHA     string     `gorm:"size:40;not null" json:"commit_sha"`
	CommitMessage string     `gorm:"type:text" json:"commit_message,omitempty"`
	CommitURL     *string    `gorm:"type:text" json:"commit_url,omitempty"`
	AuthorName    *string    `gorm:"size:255" json:"author_name,omitempty"`
	AuthorEmail   *string    `gorm:"size:255" json:"author_email,omitempty"`
	CommittedAt   *time.Time `json:"committed_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Associations
	Ticket *Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
}

func (Commit) TableName() string {
	return "ticket_commits"
}
