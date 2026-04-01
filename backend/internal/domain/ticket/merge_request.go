package ticket

import (
	"time"
)

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
