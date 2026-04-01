package ticket

import (
	"time"
)

// Ticket status constants
const (
	TicketStatusBacklog    = "backlog"
	TicketStatusTodo       = "todo"
	TicketStatusInProgress = "in_progress"
	TicketStatusInReview   = "in_review"
	TicketStatusDone       = "done"
)

// Ticket priority constants
const (
	TicketPriorityNone   = "none"
	TicketPriorityLow    = "low"
	TicketPriorityMedium = "medium"
	TicketPriorityHigh   = "high"
	TicketPriorityUrgent = "urgent"
)

// Ticket severity constants (primarily for bugs)
const (
	TicketSeverityCritical = "critical"
	TicketSeverityMajor    = "major"
	TicketSeverityMinor    = "minor"
	TicketSeverityTrivial  = "trivial"
)

// Valid estimate values (Fibonacci sequence)
var ValidEstimates = []int{1, 2, 3, 5, 8, 13, 21}

// Ticket represents a task/issue in the system
type Ticket struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	Number int    `gorm:"not null" json:"number"`
	Slug   string `gorm:"size:50;not null;uniqueIndex:idx_tickets_org_slug" json:"slug"` // e.g., "AM-123"

	Title   string  `gorm:"size:500;not null" json:"title"`
	Content *string `gorm:"type:text" json:"content,omitempty"` // Rich content (BlockNote JSON)

	Status   string  `gorm:"size:50;not null;default:'backlog';index" json:"status"`
	Priority string  `gorm:"size:50;not null;default:'none'" json:"priority"`
	Severity *string `gorm:"size:20" json:"severity,omitempty"` // For bugs: critical, major, minor, trivial
	Estimate *int    `json:"estimate,omitempty"`                 // Story points: 1, 2, 3, 5, 8, 13, 21

	DueDate     *time.Time `json:"due_date,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	RepositoryID   *int64 `gorm:"index" json:"repository_id,omitempty"`
	ReporterID     int64  `gorm:"not null" json:"reporter_id"`
	ParentTicketID *int64 `json:"parent_ticket_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Assignees     []Assignee     `gorm:"foreignKey:TicketID" json:"assignees,omitempty"`
	Labels        []Label        `gorm:"many2many:ticket_labels;" json:"labels,omitempty"`
	MergeRequests []MergeRequest `gorm:"foreignKey:TicketID" json:"merge_requests,omitempty"`
	SubTickets    []Ticket       `gorm:"foreignKey:ParentTicketID" json:"sub_tickets,omitempty"`
}

func (Ticket) TableName() string {
	return "tickets"
}

// IsActive returns true if the ticket is in an active state
func (t *Ticket) IsActive() bool {
	return t.Status == TicketStatusInProgress || t.Status == TicketStatusInReview
}

// IsCompleted returns true if the ticket is completed
func (t *Ticket) IsCompleted() bool {
	return t.Status == TicketStatusDone
}

// HasSubTickets returns true if the ticket has sub-tickets
func (t *Ticket) HasSubTickets() bool {
	return len(t.SubTickets) > 0
}

// IsValidEstimate checks if the estimate value is valid
func IsValidEstimate(estimate int) bool {
	for _, v := range ValidEstimates {
		if v == estimate {
			return true
		}
	}
	return false
}
