package ticket

import (
	"time"
)

// Ticket type constants
const (
	TicketTypeTask        = "task"
	TicketTypeBug         = "bug"
	TicketTypeFeature     = "feature"
	TicketTypeImprovement = "improvement"
	TicketTypeEpic        = "epic"
)

// Ticket status constants
const (
	TicketStatusBacklog    = "backlog"
	TicketStatusTodo       = "todo"
	TicketStatusInProgress = "in_progress"
	TicketStatusInReview   = "in_review"
	TicketStatusDone       = "done"
	TicketStatusCancelled  = "cancelled"
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

	Number     int    `gorm:"not null" json:"number"`
	Identifier string `gorm:"size:50;not null;uniqueIndex:idx_tickets_org_identifier" json:"identifier"` // e.g., "AM-123"

	Type    string  `gorm:"size:50;not null;default:'task'" json:"type"`
	Title   string  `gorm:"size:500;not null" json:"title"`
	Content *string `gorm:"type:text" json:"content,omitempty"` // Rich content (BlockNote JSON)

	Status   string `gorm:"size:50;not null;default:'backlog';index" json:"status"`
	Priority string `gorm:"size:50;not null;default:'none'" json:"priority"`
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

// IsCancelled returns true if the ticket is cancelled
func (t *Ticket) IsCancelled() bool {
	return t.Status == TicketStatusCancelled
}

// IsBug returns true if the ticket is a bug type
func (t *Ticket) IsBug() bool {
	return t.Type == TicketTypeBug
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

// Assignee represents a ticket assignee
type Assignee struct {
	TicketID int64 `gorm:"primaryKey" json:"ticket_id"`
	UserID   int64 `gorm:"primaryKey" json:"user_id"`
}

func (Assignee) TableName() string {
	return "ticket_assignees"
}

// Label represents a label that can be applied to tickets
type Label struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	RepositoryID   *int64 `gorm:"index" json:"repository_id,omitempty"` // nil = organization-level

	Name  string `gorm:"size:100;not null" json:"name"`
	Color string `gorm:"size:7;not null;default:'#6B7280'" json:"color"` // Hex color
}

func (Label) TableName() string {
	return "labels"
}

// TicketLabel represents the many-to-many relationship between tickets and labels
type TicketLabel struct {
	TicketID int64 `gorm:"primaryKey" json:"ticket_id"`
	LabelID  int64 `gorm:"primaryKey" json:"label_id"`
}

func (TicketLabel) TableName() string {
	return "ticket_labels"
}

// MR state constants
const (
	MRStateOpened = "opened"
	MRStateMerged = "merged"
	MRStateClosed = "closed"
)

// Pipeline status constants
const (
	PipelineStatusPending  = "pending"
	PipelineStatusRunning  = "running"
	PipelineStatusSuccess  = "success"
	PipelineStatusFailed   = "failed"
	PipelineStatusCanceled = "canceled"
	PipelineStatusSkipped  = "skipped"
	PipelineStatusManual   = "manual"
)

// MergeRequest represents a merge request linked to a repository
// Can optionally be associated with a ticket and/or pod
type MergeRequest struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	// Repository is required - MR always belongs to a repository
	RepositoryID int64 `gorm:"not null;index" json:"repository_id"`

	// Ticket and Pod are optional associations
	TicketID *int64 `gorm:"index" json:"ticket_id,omitempty"`
	PodID    *int64 `gorm:"index" json:"pod_id,omitempty"`

	MRIID        int    `gorm:"column:mr_iid;not null" json:"mr_iid"`
	MRURL        string `gorm:"column:mr_url;type:text;not null;uniqueIndex" json:"mr_url"`
	SourceBranch string `gorm:"size:255;not null" json:"source_branch"`
	TargetBranch string `gorm:"size:255;not null;default:'main'" json:"target_branch"`
	Title        string `gorm:"size:500" json:"title,omitempty"`
	State        string `gorm:"size:50;not null;default:'opened'" json:"state"`

	// Pipeline information
	PipelineStatus *string `gorm:"size:50" json:"pipeline_status,omitempty"`
	PipelineID     *int64  `json:"pipeline_id,omitempty"`
	PipelineURL    *string `gorm:"type:text" json:"pipeline_url,omitempty"`

	// Merge information
	MergeCommitSHA *string    `gorm:"size:40" json:"merge_commit_sha,omitempty"`
	MergedAt       *time.Time `json:"merged_at,omitempty"`
	MergedByID     *int64     `json:"merged_by_id,omitempty"`

	// Sync tracking
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Ticket *Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
}

func (MergeRequest) TableName() string {
	return "ticket_merge_requests"
}

// IsMerged returns true if the MR is merged
func (mr *MergeRequest) IsMerged() bool {
	return mr.State == MRStateMerged
}

// IsOpen returns true if the MR is open
func (mr *MergeRequest) IsOpen() bool {
	return mr.State == MRStateOpened
}

// HasPipeline returns true if the MR has a pipeline
func (mr *MergeRequest) HasPipeline() bool {
	return mr.PipelineStatus != nil
}

// IsPipelineSuccess returns true if the pipeline succeeded
func (mr *MergeRequest) IsPipelineSuccess() bool {
	return mr.PipelineStatus != nil && *mr.PipelineStatus == PipelineStatusSuccess
}

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

// Commit represents a git commit linked to a ticket
type Commit struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	TicketID     int64  `gorm:"not null;index" json:"ticket_id"`
	RepositoryID int64  `gorm:"not null;index" json:"repository_id"`
	PodID        *int64 `json:"pod_id,omitempty"`

	CommitSHA     string  `gorm:"size:40;not null" json:"commit_sha"`
	CommitMessage string  `gorm:"type:text" json:"commit_message,omitempty"`
	CommitURL     *string `gorm:"type:text" json:"commit_url,omitempty"`
	AuthorName    *string `gorm:"size:255" json:"author_name,omitempty"`
	AuthorEmail   *string `gorm:"size:255" json:"author_email,omitempty"`
	CommittedAt   *time.Time `json:"committed_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Associations
	Ticket *Ticket `gorm:"foreignKey:TicketID" json:"ticket,omitempty"`
}

func (Commit) TableName() string {
	return "ticket_commits"
}

// BoardColumn represents a kanban board column
type BoardColumn struct {
	Status string    `json:"status"`
	Count  int       `json:"count"`
	Tickets []Ticket `json:"tickets"`
}

// Board represents a kanban board view
type Board struct {
	Columns []BoardColumn `json:"columns"`
}
