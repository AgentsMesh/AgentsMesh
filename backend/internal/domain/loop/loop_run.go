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

// Trigger type constants for LoopRun (records how the run was triggered)
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

	// Run identification
	RunNumber int `gorm:"not null" json:"run_number"`

	// Status — only authoritative when pod_key is NULL (pending/skipped/failed).
	// When pod_key is set, effective status is derived from Pod via ResolveRunStatus().
	Status string `gorm:"size:20;not null;default:'pending'" json:"status"`

	// Associated resources (references to SSOT)
	PodKey                 *string `gorm:"size:100" json:"pod_key,omitempty"`
	AutopilotControllerKey *string `gorm:"size:100" json:"autopilot_controller_key,omitempty"`

	// Trigger info: how this run was triggered (cron/api/manual)
	TriggerType   string  `gorm:"size:20;not null" json:"trigger_type"`
	TriggerSource *string `gorm:"size:255" json:"trigger_source,omitempty"`

	// Runtime variables passed at trigger time (override prompt_variables defaults)
	TriggerParams json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"trigger_params,omitempty"`

	// Resolved prompt (the actual prompt sent to the agent)
	ResolvedPrompt *string `gorm:"type:text" json:"resolved_prompt,omitempty"`

	// Timing — StartedAt is set by LoopOrchestrator; FinishedAt and DurationSec
	// are derived from Pod when resolving status.
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	DurationSec *int       `json:"duration_sec,omitempty"`

	// Results — only used when pod_key is NULL (e.g., Pod creation failure).
	// When Pod exists, results come from Pod/Autopilot.
	ExitSummary  *string `gorm:"type:text" json:"exit_summary,omitempty"`
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Loop *Loop `gorm:"foreignKey:LoopID" json:"loop,omitempty"`
}

func (LoopRun) TableName() string {
	return "loop_runs"
}

// IsTerminal returns true if the run is in a terminal state.
// Note: for runs with pod_key, call ResolveRunStatus first to get the effective status.
func (r *LoopRun) IsTerminal() bool {
	return r.Status == RunStatusCompleted ||
		r.Status == RunStatusFailed ||
		r.Status == RunStatusTimeout ||
		r.Status == RunStatusCancelled ||
		r.Status == RunStatusSkipped
}

// IsActive returns true if the run is currently active.
// Note: for runs with pod_key, call ResolveRunStatus first to get the effective status.
func (r *LoopRun) IsActive() bool {
	return r.Status == RunStatusPending || r.Status == RunStatusRunning
}
