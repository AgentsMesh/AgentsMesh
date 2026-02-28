package loop

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// runRepository implements LoopRunRepository
type runRepository struct {
	db *gorm.DB
}

// NewLoopRunRepository creates a new loop run repository
func NewLoopRunRepository(db *gorm.DB) LoopRunRepository {
	return &runRepository{db: db}
}

func (r *runRepository) Create(ctx context.Context, run *LoopRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *runRepository) GetByID(ctx context.Context, id int64) (*LoopRun, error) {
	var run LoopRun
	if err := r.db.WithContext(ctx).First(&run, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *runRepository) List(ctx context.Context, filter *RunListFilter) ([]*LoopRun, int64, error) {
	query := r.db.WithContext(ctx).Where("loop_id = ?", filter.LoopID)

	// For finished runs, status in DB is authoritative — filter at DB level.
	// For active runs (pending/running), status may be resolved from Pod later,
	// so we include them regardless and let the service layer post-filter.
	if filter.Status != "" {
		query = query.Where(
			"(finished_at IS NOT NULL AND status = ?) OR (finished_at IS NULL)",
			filter.Status,
		)
	}

	var total int64
	if err := query.Model(&LoopRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	var runs []*LoopRun
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&runs).Error; err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

func (r *runRepository) Update(ctx context.Context, runID int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&LoopRun{}).
		Where("id = ?", runID).
		Updates(updates).Error
}

// FinishRun atomically marks a run as finished with optimistic locking.
// Uses WHERE finished_at IS NULL to prevent double-processing from concurrent events.
// Returns true if the row was updated, false if already finished (no-op).
func (r *runRepository) FinishRun(ctx context.Context, runID int64, updates map[string]interface{}) (bool, error) {
	updates["updated_at"] = time.Now()
	result := r.db.WithContext(ctx).
		Model(&LoopRun{}).
		Where("id = ? AND finished_at IS NULL", runID).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *runRepository) GetMaxRunNumber(ctx context.Context, loopID int64) (int, error) {
	var maxNumber int
	err := r.db.WithContext(ctx).
		Model(&LoopRun{}).
		Where("loop_id = ?", loopID).
		Select("COALESCE(MAX(run_number), 0)").
		Scan(&maxNumber).Error
	return maxNumber, err
}

func (r *runRepository) GetByAutopilotKey(ctx context.Context, autopilotKey string) (*LoopRun, error) {
	var run LoopRun
	if err := r.db.WithContext(ctx).
		Where("autopilot_controller_key = ? AND finished_at IS NULL", autopilotKey).
		First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// CountActiveRuns counts runs that are actually active, using Pod status as SSOT.
//
// A run is active if:
//   - pod_key is NULL AND status = 'pending' (Pod not yet created)
//   - pod_key is set AND the Pod is in an active state
//
// Terminated/completed Pods automatically free the concurrency slot.
func (r *runRepository) CountActiveRuns(ctx context.Context, loopID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.loop_id = ?", loopID).
		Where(
			"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
				"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
			RunStatusPending,
			agentpod.ActiveStatuses(),
		).
		Count(&count).Error
	return count, err
}

// GetActiveRunByPodKey finds an unfinished run by its pod key.
//
// Uses finished_at IS NULL as the guard instead of Pod status, because by the
// time a Pod termination event fires, the Pod is already in a terminal state.
// HandleRunCompleted sets finished_at to provide idempotency (prevents double processing).
func (r *runRepository) GetActiveRunByPodKey(ctx context.Context, podKey string) (*LoopRun, error) {
	var run LoopRun
	err := r.db.WithContext(ctx).
		Where("pod_key = ? AND finished_at IS NULL", podKey).
		First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// GetTimedOutRuns returns running runs that have exceeded their timeout.
func (r *runRepository) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error) {
	var runs []*LoopRun
	// Active statuses excluding "disconnected" — a disconnected pod
	// should not be considered "timed out" since it may reconnect.
	timedOutEligible := []string{agentpod.StatusInitializing, agentpod.StatusRunning, agentpod.StatusPaused}
	query := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("JOIN loops ON loops.id = loop_runs.loop_id").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.pod_key IS NOT NULL").
		Where("loop_runs.finished_at IS NULL").
		Where("pods.status IN ?", timedOutEligible).
		Where("loop_runs.started_at IS NOT NULL AND loop_runs.started_at < NOW() - (loops.timeout_minutes || ' minutes')::INTERVAL")
	if len(orgIDs) > 0 {
		query = query.Where("loop_runs.organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}

// ComputeLoopStats computes run statistics from Pod status (SSOT).
//
// Optimized two-phase approach to avoid loading all runs into memory:
// 1. Finished runs (finished_at IS NOT NULL): counted via SQL aggregation — their
//    status in the DB is already authoritative (set by HandleRunCompleted/MarkRunTerminal).
// 2. Active runs (finished_at IS NULL, pod_key IS NOT NULL): fetched individually
//    and resolved via Go-side DeriveRunStatus for SSOT consistency.
func (r *runRepository) ComputeLoopStats(ctx context.Context, loopID int64) (total, successful, failed int, err error) {
	// Phase 1: Aggregate finished runs via SQL (O(1) memory, efficient for large histories)
	type finishedStats struct {
		Total      int `gorm:"column:total"`
		Successful int `gorm:"column:successful"`
		Failed     int `gorm:"column:failed"`
	}
	var fs finishedStats
	err = r.db.WithContext(ctx).
		Table("loop_runs").
		Select(`
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = ?) as successful,
			COUNT(*) FILTER (WHERE status IN (?, ?, ?)) as failed
		`, RunStatusCompleted, RunStatusFailed, RunStatusTimeout, RunStatusCancelled).
		Where("loop_id = ? AND finished_at IS NOT NULL", loopID).
		Scan(&fs).Error
	if err != nil {
		return
	}
	total = fs.Total
	successful = fs.Successful
	failed = fs.Failed

	// Phase 2: Resolve active runs via Go-side SSOT (small set — bounded by max_concurrent_runs)
	type activeRunRow struct {
		Status         string  `gorm:"column:status"`
		PodKey         *string `gorm:"column:pod_key"`
		PodStatus      *string `gorm:"column:pod_status"`
		AutopilotPhase *string `gorm:"column:autopilot_phase"`
	}
	var activeRows []activeRunRow
	err = r.db.WithContext(ctx).
		Table("loop_runs lr").
		Select("lr.status, lr.pod_key, p.status as pod_status, ac.phase as autopilot_phase").
		Joins("LEFT JOIN pods p ON p.pod_key = lr.pod_key").
		Joins("LEFT JOIN autopilot_controllers ac ON ac.autopilot_controller_key = lr.autopilot_controller_key").
		Where("lr.loop_id = ? AND lr.finished_at IS NULL", loopID).
		Find(&activeRows).Error
	if err != nil {
		return
	}

	for _, row := range activeRows {
		total++

		var effectiveStatus string
		if row.PodKey == nil {
			effectiveStatus = row.Status
		} else {
			podStatus := ""
			if row.PodStatus != nil {
				podStatus = *row.PodStatus
			}
			autopilotPhase := ""
			if row.AutopilotPhase != nil {
				autopilotPhase = *row.AutopilotPhase
			}
			effectiveStatus = DeriveRunStatus(podStatus, autopilotPhase)
		}

		switch effectiveStatus {
		case RunStatusCompleted:
			successful++
		case RunStatusFailed, RunStatusTimeout, RunStatusCancelled:
			failed++
		}
	}
	return
}

// GetLatestPodKey returns the pod_key from the most recent run that has one.
func (r *runRepository) GetLatestPodKey(ctx context.Context, loopID int64) *string {
	type result struct {
		PodKey string `gorm:"column:pod_key"`
	}
	var res result
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Select("loop_runs.pod_key").
		Where("loop_runs.loop_id = ? AND loop_runs.pod_key IS NOT NULL", loopID).
		Order("loop_runs.id DESC").
		Limit(1).
		Scan(&res).Error

	if err != nil || res.PodKey == "" {
		return nil
	}
	return &res.PodKey
}

// BatchGetPodStatuses returns Pod status info for a batch of pod keys.
func (r *runRepository) BatchGetPodStatuses(ctx context.Context, podKeys []string) ([]PodStatusInfo, error) {
	if len(podKeys) == 0 {
		return nil, nil
	}

	var results []PodStatusInfo
	err := r.db.WithContext(ctx).
		Table("pods").
		Select("pod_key, status, finished_at").
		Where("pod_key IN ?", podKeys).
		Find(&results).Error
	return results, err
}

// BatchGetAutopilotPhases returns autopilot phases for a batch of keys.
func (r *runRepository) BatchGetAutopilotPhases(ctx context.Context, autopilotKeys []string) (map[string]string, error) {
	if len(autopilotKeys) == 0 {
		return nil, nil
	}

	type row struct {
		Key   string `gorm:"column:autopilot_controller_key"`
		Phase string `gorm:"column:phase"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("autopilot_controllers").
		Select("autopilot_controller_key, phase").
		Where("autopilot_controller_key IN ?", autopilotKeys).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, r := range rows {
		result[r.Key] = r.Phase
	}
	return result, nil
}

// GetOrphanPendingRuns returns pending runs with no pod_key that are stuck for > 5 minutes.
func (r *runRepository) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*LoopRun, error) {
	var runs []*LoopRun
	query := r.db.WithContext(ctx).
		Where("pod_key IS NULL").
		Where("status = ?", RunStatusPending).
		Where("finished_at IS NULL").
		Where("created_at < NOW() - INTERVAL '5 minutes'")
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}

// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops using Pod status (SSOT).
func (r *runRepository) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
	if len(loopIDs) == 0 {
		return nil, nil
	}

	type countRow struct {
		LoopID int64 `gorm:"column:loop_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []countRow
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Select("loop_runs.loop_id, COUNT(*) as count").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.loop_id IN ?", loopIDs).
		Where(
			"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
				"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
			RunStatusPending,
			agentpod.ActiveStatuses(),
		).
		Group("loop_runs.loop_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(rows))
	for _, row := range rows {
		result[row.LoopID] = row.Count
	}
	return result, nil
}

// GetAvgDuration returns the average duration in seconds for completed runs of a loop.
func (r *runRepository) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	var avg *float64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Where("loop_id = ? AND duration_sec IS NOT NULL AND finished_at IS NOT NULL", loopID).
		Select("AVG(duration_sec)").
		Scan(&avg).Error
	return avg, err
}

// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
// Keeps the most recent `keep` finished runs (by id DESC), deletes the rest.
func (r *runRepository) DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error) {
	if keep <= 0 {
		return 0, nil
	}

	// Delete finished runs that are outside the retention window.
	// Uses a subquery to find the cutoff ID — runs with id <= cutoff are deleted.
	result := r.db.WithContext(ctx).Exec(`
		DELETE FROM loop_runs
		WHERE loop_id = ? AND finished_at IS NOT NULL
		  AND id NOT IN (
		    SELECT id FROM loop_runs
		    WHERE loop_id = ? AND finished_at IS NOT NULL
		    ORDER BY id DESC
		    LIMIT ?
		  )
	`, loopID, loopID, keep)

	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// TriggerRunAtomic atomically creates a loop run within a FOR UPDATE transaction.
// Handles concurrency check (SSOT via Pod JOIN), run number generation, and record creation.
func (r *runRepository) TriggerRunAtomic(ctx context.Context, params *TriggerRunAtomicParams) (*TriggerRunAtomicResult, error) {
	var result *TriggerRunAtomicResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock the loop row with FOR UPDATE to serialize concurrent triggers
		var loop Loop
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&loop, params.LoopID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("failed to get loop: %w", err)
		}

		if !loop.IsEnabled() {
			return ErrLoopDisabled
		}

		// 2. Count active runs using Pod status (SSOT) — within the transaction
		var activeCount int64
		if err := tx.Table("loop_runs").
			Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
			Where("loop_runs.loop_id = ?", loop.ID).
			Where(
				"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
					"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
				RunStatusPending,
				agentpod.ActiveStatuses(),
			).
			Count(&activeCount).Error; err != nil {
			return fmt.Errorf("failed to count active runs: %w", err)
		}

		if activeCount >= int64(loop.MaxConcurrentRuns) {
			switch loop.ConcurrencyPolicy {
			case ConcurrencyPolicySkip:
				// Create skipped run (still in transaction for atomic run_number)
				var maxNumber int
				tx.Model(&LoopRun{}).
					Where("loop_id = ?", loop.ID).
					Select("COALESCE(MAX(run_number), 0)").
					Scan(&maxNumber)

				now := time.Now()
				skippedRun := &LoopRun{
					OrganizationID: loop.OrganizationID,
					LoopID:         loop.ID,
					RunNumber:      maxNumber + 1,
					Status:         RunStatusSkipped,
					TriggerType:    params.TriggerType,
					TriggerSource:  &params.TriggerSource,
					FinishedAt:     &now, // Mark as finished so retention cleanup can cover it
				}
				if err := tx.Create(skippedRun).Error; err != nil {
					return err
				}
				result = &TriggerRunAtomicResult{
					Run:     skippedRun,
					Loop:    &loop,
					Skipped: true,
					Reason:  "max concurrent runs reached",
				}
				return nil
			case ConcurrencyPolicyQueue:
				// Queue is not yet implemented — create a skipped run record
				var qMaxNumber int
				tx.Model(&LoopRun{}).
					Where("loop_id = ?", loop.ID).
					Select("COALESCE(MAX(run_number), 0)").
					Scan(&qMaxNumber)
				qNow := time.Now()
				queuedRun := &LoopRun{
					OrganizationID: loop.OrganizationID,
					LoopID:         loop.ID,
					RunNumber:      qMaxNumber + 1,
					Status:         RunStatusSkipped,
					TriggerType:    params.TriggerType,
					TriggerSource:  &params.TriggerSource,
					FinishedAt:     &qNow,
				}
				if err := tx.Create(queuedRun).Error; err != nil {
					return err
				}
				result = &TriggerRunAtomicResult{
					Run:     queuedRun,
					Loop:    &loop,
					Skipped: true,
					Reason:  "queued (not yet implemented)",
				}
				return nil
			case ConcurrencyPolicyReplace:
				// Replace is not yet implemented — create a skipped run record
				var rMaxNumber int
				tx.Model(&LoopRun{}).
					Where("loop_id = ?", loop.ID).
					Select("COALESCE(MAX(run_number), 0)").
					Scan(&rMaxNumber)
				rNow := time.Now()
				replacedRun := &LoopRun{
					OrganizationID: loop.OrganizationID,
					LoopID:         loop.ID,
					RunNumber:      rMaxNumber + 1,
					Status:         RunStatusSkipped,
					TriggerType:    params.TriggerType,
					TriggerSource:  &params.TriggerSource,
					FinishedAt:     &rNow,
				}
				if err := tx.Create(replacedRun).Error; err != nil {
					return err
				}
				result = &TriggerRunAtomicResult{
					Run:     replacedRun,
					Loop:    &loop,
					Skipped: true,
					Reason:  "replace (not yet implemented)",
				}
				return nil
			}
		}

		// 3. Get next run number atomically (inside transaction with lock)
		var maxNumber int
		if err := tx.Model(&LoopRun{}).
			Where("loop_id = ?", loop.ID).
			Select("COALESCE(MAX(run_number), 0)").
			Scan(&maxNumber).Error; err != nil {
			return fmt.Errorf("failed to get next run number: %w", err)
		}
		runNumber := maxNumber + 1

		// 4. Create the run record (status=pending, no pod_key yet)
		resolvedPrompt := loop.PromptTemplate
		now := time.Now()

		run := &LoopRun{
			OrganizationID: loop.OrganizationID,
			LoopID:         loop.ID,
			RunNumber:      runNumber,
			Status:         RunStatusPending,
			TriggerType:    params.TriggerType,
			TriggerSource:  &params.TriggerSource,
			TriggerParams:  params.TriggerParams,
			ResolvedPrompt: &resolvedPrompt,
			StartedAt:      &now,
		}

		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("failed to create loop run: %w", err)
		}

		result = &TriggerRunAtomicResult{Run: run, Loop: &loop}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
