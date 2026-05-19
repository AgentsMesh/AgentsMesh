package loop

import "context"

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
	GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
	// GetOrphanPendingRuns returns pending runs with no pod_key stuck for > 5 minutes.
	GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
	ComputeLoopStats(ctx context.Context, loopID int64) (total, successful, failed int, err error)
	GetLatestPodKey(ctx context.Context, loopID int64) *string

	// SSOT: batch status resolution helpers
	BatchGetPodStatuses(ctx context.Context, podKeys []string) ([]PodStatusInfo, error)
	BatchGetAutopilotPhases(ctx context.Context, autopilotKeys []string) (map[string]string, error)

	CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error)

	GetAvgDuration(ctx context.Context, loopID int64) (*float64, error)

	// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
	// Keeps the most recent `keep` finished runs, deletes the rest.
	// Returns the number of rows deleted.
	DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error)

	GetIdleLoopPods(ctx context.Context, orgIDs []int64) ([]*LoopRun, error)
}
