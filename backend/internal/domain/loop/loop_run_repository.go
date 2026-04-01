package loop

import "context"

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

	// GetIdleLoopPods returns active loop runs whose Pods have been idle (agent waiting)
	// longer than the loop's idle_timeout_sec. Used by the scheduler to auto-terminate.
	GetIdleLoopPods(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
}
