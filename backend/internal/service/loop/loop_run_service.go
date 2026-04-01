package loop

import (
	"context"
	"errors"
	"log/slog"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

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
	return &LoopRunService{repo: repo}
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
	if err := s.repo.Create(ctx, run); err != nil {
		slog.Error("failed to create loop run", "loop_id", run.LoopID, "run_number", run.RunNumber, "error", err)
		return err
	}
	slog.Info("loop run created", "run_id", run.ID, "loop_id", run.LoopID, "run_number", run.RunNumber)
	return nil
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

	s.resolveRunStatuses(ctx, runs)

	if filter.Status != "" {
		filtered := make([]*loopDomain.LoopRun, 0, len(runs))
		for _, run := range runs {
			if run.Status == filter.Status {
				filtered = append(filtered, run)
			}
		}
		removed := int64(len(runs) - len(filtered))
		runs = filtered
		total -= removed
	}

	return runs, total, nil
}

// TriggerRunAtomic atomically triggers a loop run within a FOR UPDATE transaction.
func (s *LoopRunService) TriggerRunAtomic(ctx context.Context, params *loopDomain.TriggerRunAtomicParams) (*loopDomain.TriggerRunAtomicResult, error) {
	return s.repo.TriggerRunAtomic(ctx, params)
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
func (s *LoopRunService) FinishRun(ctx context.Context, runID int64, updates map[string]interface{}) (bool, error) {
	updated, err := s.repo.FinishRun(ctx, runID, updates)
	if err != nil {
		slog.Error("failed to finish loop run", "run_id", runID, "error", err)
		return false, err
	}
	if updated {
		slog.Info("loop run finished", "run_id", runID, "status", updates["status"])
	}
	return updated, nil
}

// GetActiveRunByPodKey finds an active run by its pod key, with status resolved from Pod (SSOT).
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

