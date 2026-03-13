package loop

import (
	"context"
	"encoding/json"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
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
	// When pod_key is set, effective status is derived from Pod via ResolveStatus().
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

// ResolveStatus resolves the effective status of this run using Pod and Autopilot state.
// This implements SSOT: Pod status is the single source of truth for execution state.
//
// Parameters:
//   - podStatus:       Pod.Status (e.g., "running", "completed", "terminated")
//   - autopilotPhase:  AutopilotController.Phase (empty string if no autopilot)
//   - podFinishedAt:   Pod.FinishedAt (for computing duration)
func (r *LoopRun) ResolveStatus(podStatus string, autopilotPhase string, podFinishedAt *time.Time) {
	// No Pod → keep the run's own status (pending/skipped/failed)
	if r.PodKey == nil {
		return
	}

	// Derive status from Autopilot phase (if present) or Pod status
	r.Status = DeriveRunStatus(podStatus, autopilotPhase)

	// Derive timing from Pod
	if podFinishedAt != nil {
		r.FinishedAt = podFinishedAt
		if r.StartedAt != nil {
			d := int(podFinishedAt.Sub(*r.StartedAt).Seconds())
			r.DurationSec = &d
		}
	}
}

// DeriveRunStatus maps Pod/Autopilot state to Loop Run status.
//
// Priority logic:
//   - Autopilot terminal phase (completed/failed/stopped) is authoritative
//   - If autopilot is non-terminal but Pod is done, Pod wins (ground truth)
//   - For Direct mode (no autopilot), Pod status is used directly
//
// This handles the case where a Pod is manually terminated while autopilot
// is still in an active phase — the Pod's terminal state is the ground truth.
func DeriveRunStatus(podStatus string, autopilotPhase string) string {
	// Autopilot mode
	if autopilotPhase != "" {
		// Autopilot terminal phases are authoritative
		switch autopilotPhase {
		case agentpod.AutopilotPhaseCompleted:
			return RunStatusCompleted
		case agentpod.AutopilotPhaseFailed:
			return RunStatusFailed
		case agentpod.AutopilotPhaseStopped:
			return RunStatusCancelled
		case agentpod.AutopilotPhaseMaxIterations:
			// max_iterations means "task not finished but iteration quota exhausted".
			// Map to completed — the autopilot did its best within the configured limit.
			return RunStatusCompleted
		default:
			// Autopilot is in active phase — but if Pod is done,
			// Pod's state is the ground truth (SSOT)
			if isPodDoneForLoop(podStatus) {
				return deriveFromPodStatus(podStatus)
			}
			return RunStatusRunning
		}
	}

	// Direct mode: Pod status is the truth
	if isPodDoneForLoop(podStatus) {
		return deriveFromPodStatus(podStatus)
	}
	return RunStatusRunning
}

// isPodDoneForLoop returns true if the Pod is "done" from the Loop domain's perspective.
//
// This deliberately excludes StatusOrphaned — an orphaned pod may reconnect,
// so from Loop's perspective it's still potentially active.
// This differs from Pod.IsTerminal() which includes orphaned.
func isPodDoneForLoop(podStatus string) bool {
	return podStatus == agentpod.StatusCompleted ||
		podStatus == agentpod.StatusTerminated ||
		podStatus == agentpod.StatusError
}

// deriveFromPodStatus maps a "done" Pod status to Loop Run status.
//
// StatusCompleted = Pod process exited naturally → run completed successfully.
// StatusTerminated = Pod was explicitly killed (user cancel, system cleanup) → run cancelled.
// StatusError = Pod encountered an error → run failed.
func deriveFromPodStatus(podStatus string) string {
	switch podStatus {
	case agentpod.StatusCompleted:
		return RunStatusCompleted
	case agentpod.StatusTerminated:
		return RunStatusCancelled
	case agentpod.StatusError:
		return RunStatusFailed
	default:
		return RunStatusFailed
	}
}

// IsTerminal returns true if the run is in a terminal state.
// Note: for runs with pod_key, call ResolveStatus first to get the effective status.
func (r *LoopRun) IsTerminal() bool {
	return r.Status == RunStatusCompleted ||
		r.Status == RunStatusFailed ||
		r.Status == RunStatusTimeout ||
		r.Status == RunStatusCancelled ||
		r.Status == RunStatusSkipped
}

// IsActive returns true if the run is currently active.
// Note: for runs with pod_key, call ResolveStatus first to get the effective status.
func (r *LoopRun) IsActive() bool {
	return r.Status == RunStatusPending || r.Status == RunStatusRunning
}

// PodStatusInfo holds Pod status info for SSOT resolution
type PodStatusInfo struct {
	PodKey     string
	Status     string
	FinishedAt *time.Time
}

// RunListFilter represents filters for listing loop runs
type RunListFilter struct {
	LoopID int64
	Status string // Optional: filter by status (applied at DB level for finished runs)
	Limit  int
	Offset int
}

// TriggerRunAtomicParams contains parameters for atomically creating a loop run.
type TriggerRunAtomicParams struct {
	LoopID        int64
	TriggerType   string
	TriggerSource string
	TriggerParams json.RawMessage // Optional runtime variable overrides
}

// TriggerRunAtomicResult contains the result of an atomic trigger operation.
type TriggerRunAtomicResult struct {
	Run     *LoopRun
	Loop    *Loop // the loop as read within the transaction (for event publishing)
	Skipped bool
	Reason  string
}

// LoopRunRepository defines the interface for loop run data access
type LoopRunRepository interface {
	Create(ctx context.Context, run *LoopRun) error
	GetByID(ctx context.Context, id int64) (*LoopRun, error)
	List(ctx context.Context, filter *RunListFilter) ([]*LoopRun, int64, error)
	Update(ctx context.Context, runID int64, updates map[string]interface{}) error
	GetMaxRunNumber(ctx context.Context, loopID int64) (int, error)
	GetByAutopilotKey(ctx context.Context, autopilotKey string) (*LoopRun, error)

	// TriggerRunAtomic atomically creates a loop run within a FOR UPDATE transaction.
	// Handles concurrency check (SSOT via Pod JOIN), run number generation, and record creation.
	TriggerRunAtomic(ctx context.Context, params *TriggerRunAtomicParams) (*TriggerRunAtomicResult, error)

	// FinishRun atomically marks a run as finished with optimistic locking.
	// Uses WHERE finished_at IS NULL to prevent double-processing from concurrent events.
	// Returns true if the row was updated (caller should proceed), false if already finished.
	FinishRun(ctx context.Context, runID int64, updates map[string]interface{}) (bool, error)

	// SSOT: cross-table queries (JOIN with pods/autopilot_controllers)
	CountActiveRuns(ctx context.Context, loopID int64) (int64, error)
	GetActiveRunByPodKey(ctx context.Context, podKey string) (*LoopRun, error)
	// GetTimedOutRuns returns running runs that have exceeded their timeout.
	// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
	GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
	// GetOrphanPendingRuns returns pending runs with no pod_key stuck for > 5 minutes.
	GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
	ComputeLoopStats(ctx context.Context, loopID int64) (total, successful, failed int, err error)
	GetLatestPodKey(ctx context.Context, loopID int64) *string

	// SSOT: batch status resolution helpers
	BatchGetPodStatuses(ctx context.Context, podKeys []string) ([]PodStatusInfo, error)
	BatchGetAutopilotPhases(ctx context.Context, autopilotKeys []string) (map[string]string, error)

	// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops.
	CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error)

	// GetAvgDuration returns the average duration in seconds for completed runs of a loop.
	GetAvgDuration(ctx context.Context, loopID int64) (*float64, error)

	// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
	// Keeps the most recent `keep` finished runs, deletes the rest.
	// Returns the number of rows deleted.
	DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error)
}
