package loop

import (
	"encoding/json"
	"time"
)

// LoopRun status constants
//
// Status lifecycle:
//   - "pending": initial state before Pod is created
//   - "skipped": concurrency policy prevented execution (terminal)
//   - "failed":  Pod creation failed, no Pod exists (terminal)
//
// Once pod_key is set, the run's effective status is DERIVED from Pod status.
// The status field in DB is NOT updated after pod_key is set — Pod is the
// Single Source of Truth (SSOT) for execution state.
const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusTimeout   = "timeout"
	RunStatusCancelled = "cancelled"
	RunStatusSkipped   = "skipped"
)

const (
	RunTriggerCron   = "cron"
	RunTriggerAPI    = "api"
	RunTriggerManual = "manual"
)

// LoopRun represents a single execution record of a Loop.
//
// The run's effective status follows SSOT: once a Pod is associated (pod_key is set),
// the status is derived from the Pod's status — never maintained independently.
type LoopRun struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`
	LoopID         int64 `gorm:"not null;index" json:"loop_id"`

	RunNumber int `gorm:"not null" json:"run_number"`

	Status string `gorm:"size:20;not null;default:'pending'" json:"status"`

	// Associated resources (references to SSOT)
	PodKey                 *string `gorm:"size:100" json:"pod_key,omitempty"`
	AutopilotControllerKey *string `gorm:"size:100" json:"autopilot_controller_key,omitempty"`

	TriggerType   string  `gorm:"size:20;not null" json:"trigger_type"`
	TriggerSource *string `gorm:"size:255" json:"trigger_source,omitempty"`

	TriggerParams json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"trigger_params,omitempty"`

	ResolvedPrompt *string `gorm:"type:text" json:"resolved_prompt,omitempty"`

	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	DurationSec *int       `json:"duration_sec,omitempty"`

	ExitSummary  *string `gorm:"type:text" json:"exit_summary,omitempty"`
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Loop *Loop `gorm:"foreignKey:LoopID" json:"loop,omitempty"`
}

func (LoopRun) TableName() string {
	return "loop_runs"
}

func (r *LoopRun) IsTerminal() bool {
	return r.Status == RunStatusCompleted ||
		r.Status == RunStatusFailed ||
		r.Status == RunStatusTimeout ||
		r.Status == RunStatusCancelled ||
		r.Status == RunStatusSkipped
}

func (r *LoopRun) IsActive() bool {
	return r.Status == RunStatusPending || r.Status == RunStatusRunning
}
