package loop

import (
	"context"
	"errors"
	"log/slog"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// TriggerRunAtomic atomically triggers a loop run within a FOR UPDATE transaction.
// Delegates to the repository for transactional safety.
func (s *LoopRunService) TriggerRunAtomic(ctx context.Context, params *loopDomain.TriggerRunAtomicParams) (*loopDomain.TriggerRunAtomicResult, error) {
	return s.repo.TriggerRunAtomic(ctx, params)
}

var (
	ErrRunNotFound = errors.New("loop run not found")
)

// LoopRunService handles LoopRun operations.
//
// All read methods resolve the effective status from Pod (SSOT).
// The run's own status field is only authoritative when pod_key is NULL.
type LoopRunService struct {
	repo loopDomain.LoopRunRepository
}

// NewLoopRunService creates a new LoopRunService
func NewLoopRunService(repo loopDomain.LoopRunRepository) *LoopRunService {
	return &LoopRunService{
		repo: repo,
	}
}

// ListRunsFilter represents filters for listing loop runs (service-level)
type ListRunsFilter struct {
	LoopID int64
	Status string
	Limit  int
	Offset int
}

// Create creates a new LoopRun
func (s *LoopRunService) Create(ctx context.Context, run *loopDomain.LoopRun) error {
	return s.repo.Create(ctx, run)
}

// GetByID retrieves a LoopRun by ID, with status resolved from Pod (SSOT).
func (s *LoopRunService) GetByID(ctx context.Context, id int64) (*loopDomain.LoopRun, error) {
	run, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	s.resolveRunStatus(ctx, run)
	return run, nil
}

// ListRuns lists LoopRuns with filters, with statuses resolved from Pod (SSOT).
//
// Status filtering is two-phase:
//  1. DB-level: finished runs (finished_at IS NOT NULL) are filtered by persisted status
//  2. Post-resolution: active runs are resolved from Pod SSOT, then filtered
//
// This ensures pagination works correctly for finished runs (the vast majority).
func (s *LoopRunService) ListRuns(ctx context.Context, filter *ListRunsFilter) ([]*loopDomain.LoopRun, int64, error) {
	runs, total, err := s.repo.List(ctx, &loopDomain.RunListFilter{
		LoopID: filter.LoopID,
		Status: filter.Status,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
	if err != nil {
		return nil, 0, err
	}

	// Resolve statuses from Pod (SSOT) for active runs
	s.resolveRunStatuses(ctx, runs)

	// Post-filter active runs whose resolved status doesn't match the filter
	if filter.Status != "" {
		filtered := make([]*loopDomain.LoopRun, 0, len(runs))
		for _, run := range runs {
			if run.Status == filter.Status {
				filtered = append(filtered, run)
			}
		}
		// Only adjust total if active runs were filtered out
		removed := int64(len(runs) - len(filtered))
		runs = filtered
		total -= removed
	}

	return runs, total, nil
}

// GetNextRunNumber returns the next run number for a loop
func (s *LoopRunService) GetNextRunNumber(ctx context.Context, loopID int64) (int, error) {
	maxNumber, err := s.repo.GetMaxRunNumber(ctx, loopID)
	if err != nil {
		return 0, err
	}
	return maxNumber + 1, nil
}

// CountActiveRuns counts runs that are actually active, using Pod status as SSOT.
func (s *LoopRunService) CountActiveRuns(ctx context.Context, loopID int64) (int64, error) {
	return s.repo.CountActiveRuns(ctx, loopID)
}

// UpdateStatus updates fields on a LoopRun (used for pre-Pod states only).
func (s *LoopRunService) UpdateStatus(ctx context.Context, runID int64, updates map[string]interface{}) error {
	return s.repo.Update(ctx, runID, updates)
}

// FinishRun atomically marks a run as finished with optimistic locking.
// Returns true if the row was updated (caller should proceed with side-effects),
// false if already finished by a concurrent event (caller should skip).
func (s *LoopRunService) FinishRun(ctx context.Context, runID int64, updates map[string]interface{}) (bool, error) {
	return s.repo.FinishRun(ctx, runID, updates)
}

// GetActiveRunByPodKey finds an active run by its pod key, with status resolved from Pod (SSOT).
// Use this for API consumers that need the effective status.
func (s *LoopRunService) GetActiveRunByPodKey(ctx context.Context, podKey string) (*loopDomain.LoopRun, error) {
	run, err := s.repo.GetActiveRunByPodKey(ctx, podKey)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	s.resolveRunStatus(ctx, run)
	return run, nil
}

// FindActiveRunByPodKey finds an active run by its pod key WITHOUT resolving status.
// Use this for internal event handlers that derive status from the event payload
// (avoids redundant Pod status queries).
func (s *LoopRunService) FindActiveRunByPodKey(ctx context.Context, podKey string) (*loopDomain.LoopRun, error) {
	run, err := s.repo.GetActiveRunByPodKey(ctx, podKey)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}
	return run, nil
}

// GetActiveRunByAutopilotKey finds an active run by its autopilot controller key, with status resolved.
// Use this for API consumers that need the effective status.
func (s *LoopRunService) GetActiveRunByAutopilotKey(ctx context.Context, autopilotKey string) (*loopDomain.LoopRun, error) {
	run, err := s.repo.GetByAutopilotKey(ctx, autopilotKey)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}

	s.resolveRunStatus(ctx, run)
	return run, nil
}

// FindActiveRunByAutopilotKey finds an active run by its autopilot controller key WITHOUT resolving status.
// Use this for internal event handlers that derive status from the event payload.
func (s *LoopRunService) FindActiveRunByAutopilotKey(ctx context.Context, autopilotKey string) (*loopDomain.LoopRun, error) {
	run, err := s.repo.GetByAutopilotKey(ctx, autopilotKey)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrRunNotFound
		}
		return nil, err
	}
	return run, nil
}

// GetTimedOutRuns returns running runs that have exceeded their timeout.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (s *LoopRunService) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetTimedOutRuns(ctx, orgIDs)
}

// GetOrphanPendingRuns returns pending runs with no pod_key that are stuck for > 5 minutes.
// These occur when StartRun goroutine fails or server restarts between TriggerRun and StartRun.
func (s *LoopRunService) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetOrphanPendingRuns(ctx, orgIDs)
}

// ComputeLoopStats computes run statistics from Pod status (SSOT).
func (s *LoopRunService) ComputeLoopStats(ctx context.Context, loopID int64) (total int, successful int, failed int, err error) {
	return s.repo.ComputeLoopStats(ctx, loopID)
}

// GetLatestPodKey returns the pod_key from the most recent run that has one.
func (s *LoopRunService) GetLatestPodKey(ctx context.Context, loopID int64) *string {
	return s.repo.GetLatestPodKey(ctx, loopID)
}

// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops.
func (s *LoopRunService) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
	return s.repo.CountActiveRunsByLoopIDs(ctx, loopIDs)
}

// GetAvgDuration returns the average duration in seconds for a loop.
func (s *LoopRunService) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	return s.repo.GetAvgDuration(ctx, loopID)
}

// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
func (s *LoopRunService) DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error) {
	return s.repo.DeleteOldFinishedRuns(ctx, loopID, keep)
}

// GetAutopilotPhase returns the autopilot phase for a given controller key.
func (s *LoopRunService) GetAutopilotPhase(ctx context.Context, autopilotKey string) string {
	phases, err := s.repo.BatchGetAutopilotPhases(ctx, []string{autopilotKey})
	if err != nil || phases == nil {
		return ""
	}
	return phases[autopilotKey]
}

// resolveRunStatuses resolves status for a batch of runs from Pod (SSOT).
func (s *LoopRunService) resolveRunStatuses(ctx context.Context, runs []*loopDomain.LoopRun) {
	// Collect pod keys and autopilot keys
	podKeys := make([]string, 0)
	autopilotKeys := make([]string, 0)
	for _, run := range runs {
		if run.PodKey != nil {
			podKeys = append(podKeys, *run.PodKey)
		}
		if run.AutopilotControllerKey != nil {
			autopilotKeys = append(autopilotKeys, *run.AutopilotControllerKey)
		}
	}

	if len(podKeys) == 0 {
		return
	}

	// Batch query Pod statuses via repository
	podInfos, err := s.repo.BatchGetPodStatuses(ctx, podKeys)
	if err != nil {
		slog.Error("failed to batch get pod statuses for SSOT resolution", "error", err, "count", len(podKeys))
	}
	podMap := make(map[string]*loopDomain.PodStatusInfo, len(podInfos))
	for i := range podInfos {
		podMap[podInfos[i].PodKey] = &podInfos[i]
	}

	// Batch query Autopilot phases via repository
	autopilotMap, err := s.repo.BatchGetAutopilotPhases(ctx, autopilotKeys)
	if err != nil {
		slog.Error("failed to batch get autopilot phases for SSOT resolution", "error", err, "count", len(autopilotKeys))
	}

	// Resolve each run using domain logic
	for _, run := range runs {
		if run.PodKey == nil {
			continue
		}
		// If finished_at is set, the run has been finalized by HandleRunCompleted.
		// Use the persisted status/duration directly instead of re-deriving from Pod.
		// This avoids stale/missing Pod data causing incorrect status after completion.
		if run.FinishedAt != nil {
			continue
		}
		pod, ok := podMap[*run.PodKey]
		if !ok {
			// Pod not found in DB — treat as failed (orphaned reference)
			run.Status = loopDomain.RunStatusFailed
			continue
		}

		autopilotPhase := ""
		if run.AutopilotControllerKey != nil && autopilotMap != nil {
			autopilotPhase = autopilotMap[*run.AutopilotControllerKey]
		}

		run.ResolveStatus(pod.Status, autopilotPhase, pod.FinishedAt)
	}
}

// resolveRunStatus resolves status for a single run from Pod (SSOT).
func (s *LoopRunService) resolveRunStatus(ctx context.Context, run *loopDomain.LoopRun) {
	if run.PodKey == nil {
		return
	}
	s.resolveRunStatuses(ctx, []*loopDomain.LoopRun{run})
}
