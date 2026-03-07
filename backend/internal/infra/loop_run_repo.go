package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// loopRunRepo implements loop.LoopRunRepository using GORM
type loopRunRepo struct {
	db *gorm.DB
}

// NewLoopRunRepository creates a new loop run repository
func NewLoopRunRepository(db *gorm.DB) loop.LoopRunRepository {
	return &loopRunRepo{db: db}
}

func (r *loopRunRepo) Create(ctx context.Context, run *loop.LoopRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *loopRunRepo) GetByID(ctx context.Context, id int64) (*loop.LoopRun, error) {
	var run loop.LoopRun
	if err := r.db.WithContext(ctx).First(&run, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

func (r *loopRunRepo) List(ctx context.Context, filter *loop.RunListFilter) ([]*loop.LoopRun, int64, error) {
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
	if err := query.Model(&loop.LoopRun{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	var runs []*loop.LoopRun
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&runs).Error; err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

func (r *loopRunRepo) Update(ctx context.Context, runID int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&loop.LoopRun{}).
		Where("id = ?", runID).
		Updates(updates).Error
}

// FinishRun atomically marks a run as finished with optimistic locking.
func (r *loopRunRepo) FinishRun(ctx context.Context, runID int64, updates map[string]interface{}) (bool, error) {
	updates["updated_at"] = time.Now()
	result := r.db.WithContext(ctx).
		Model(&loop.LoopRun{}).
		Where("id = ? AND finished_at IS NULL", runID).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *loopRunRepo) GetMaxRunNumber(ctx context.Context, loopID int64) (int, error) {
	var maxNumber int
	err := r.db.WithContext(ctx).
		Model(&loop.LoopRun{}).
		Where("loop_id = ?", loopID).
		Select("COALESCE(MAX(run_number), 0)").
		Scan(&maxNumber).Error
	return maxNumber, err
}

func (r *loopRunRepo) GetByAutopilotKey(ctx context.Context, autopilotKey string) (*loop.LoopRun, error) {
	var run loop.LoopRun
	if err := r.db.WithContext(ctx).
		Where("autopilot_controller_key = ? AND finished_at IS NULL", autopilotKey).
		First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// CountActiveRuns counts runs that are actually active, using Pod status as SSOT.
func (r *loopRunRepo) CountActiveRuns(ctx context.Context, loopID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.loop_id = ?", loopID).
		Where(
			"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
				"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
			loop.RunStatusPending,
			agentpod.ActiveStatuses(),
		).
		Count(&count).Error
	return count, err
}

// GetActiveRunByPodKey finds an unfinished run by its pod key.
func (r *loopRunRepo) GetActiveRunByPodKey(ctx context.Context, podKey string) (*loop.LoopRun, error) {
	var run loop.LoopRun
	err := r.db.WithContext(ctx).
		Where("pod_key = ? AND finished_at IS NULL", podKey).
		First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// GetTimedOutRuns returns running runs that have exceeded their timeout.
func (r *loopRunRepo) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*loop.LoopRun, error) {
	var runs []*loop.LoopRun
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
func (r *loopRunRepo) ComputeLoopStats(ctx context.Context, loopID int64) (total, successful, failed int, err error) {
	// Phase 1: Aggregate finished runs via SQL
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
		`, loop.RunStatusCompleted, loop.RunStatusFailed, loop.RunStatusTimeout, loop.RunStatusCancelled).
		Where("loop_id = ? AND finished_at IS NOT NULL", loopID).
		Scan(&fs).Error
	if err != nil {
		return
	}
	total = fs.Total
	successful = fs.Successful
	failed = fs.Failed

	// Phase 2: Resolve active runs via Go-side SSOT
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
			effectiveStatus = loop.DeriveRunStatus(podStatus, autopilotPhase)
		}

		switch effectiveStatus {
		case loop.RunStatusCompleted:
			successful++
		case loop.RunStatusFailed, loop.RunStatusTimeout, loop.RunStatusCancelled:
			failed++
		}
	}
	return
}

// GetLatestPodKey returns the pod_key from the most recent run that has one.
func (r *loopRunRepo) GetLatestPodKey(ctx context.Context, loopID int64) *string {
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
func (r *loopRunRepo) BatchGetPodStatuses(ctx context.Context, podKeys []string) ([]loop.PodStatusInfo, error) {
	if len(podKeys) == 0 {
		return nil, nil
	}

	var results []loop.PodStatusInfo
	err := r.db.WithContext(ctx).
		Table("pods").
		Select("pod_key, status, finished_at").
		Where("pod_key IN ?", podKeys).
		Find(&results).Error
	return results, err
}

// BatchGetAutopilotPhases returns autopilot phases for a batch of keys.
func (r *loopRunRepo) BatchGetAutopilotPhases(ctx context.Context, autopilotKeys []string) (map[string]string, error) {
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
func (r *loopRunRepo) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*loop.LoopRun, error) {
	var runs []*loop.LoopRun
	query := r.db.WithContext(ctx).
		Where("pod_key IS NULL").
		Where("status = ?", loop.RunStatusPending).
		Where("finished_at IS NULL").
		Where("created_at < NOW() - INTERVAL '5 minutes'")
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}

// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops using Pod status (SSOT).
func (r *loopRunRepo) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
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
			loop.RunStatusPending,
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
func (r *loopRunRepo) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	var avg *float64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Where("loop_id = ? AND duration_sec IS NOT NULL AND finished_at IS NOT NULL", loopID).
		Select("AVG(duration_sec)").
		Scan(&avg).Error
	return avg, err
}

// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
func (r *loopRunRepo) DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error) {
	if keep <= 0 {
		return 0, nil
	}

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
func (r *loopRunRepo) TriggerRunAtomic(ctx context.Context, params *loop.TriggerRunAtomicParams) (*loop.TriggerRunAtomicResult, error) {
	var result *loop.TriggerRunAtomicResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock the loop row with FOR UPDATE to serialize concurrent triggers
		var l loop.Loop
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&l, params.LoopID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return loop.ErrNotFound
			}
			return fmt.Errorf("failed to get loop: %w", err)
		}

		if !l.IsEnabled() {
			return loop.ErrLoopDisabled
		}

		// 2. Count active runs using Pod status (SSOT) — within the transaction
		var activeCount int64
		if err := tx.Table("loop_runs").
			Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
			Where("loop_runs.loop_id = ?", l.ID).
			Where(
				"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
					"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
				loop.RunStatusPending,
				agentpod.ActiveStatuses(),
			).
			Count(&activeCount).Error; err != nil {
			return fmt.Errorf("failed to count active runs: %w", err)
		}

		if activeCount >= int64(l.MaxConcurrentRuns) {
			return r.handleConcurrencyPolicy(tx, &l, params, &result)
		}

		// 3. Get next run number atomically (inside transaction with lock)
		var maxNumber int
		if err := tx.Model(&loop.LoopRun{}).
			Where("loop_id = ?", l.ID).
			Select("COALESCE(MAX(run_number), 0)").
			Scan(&maxNumber).Error; err != nil {
			return fmt.Errorf("failed to get next run number: %w", err)
		}
		runNumber := maxNumber + 1

		// 4. Create the run record (status=pending, no pod_key yet)
		resolvedPrompt := l.PromptTemplate
		now := time.Now()

		run := &loop.LoopRun{
			OrganizationID: l.OrganizationID,
			LoopID:         l.ID,
			RunNumber:      runNumber,
			Status:         loop.RunStatusPending,
			TriggerType:    params.TriggerType,
			TriggerSource:  &params.TriggerSource,
			TriggerParams:  params.TriggerParams,
			ResolvedPrompt: &resolvedPrompt,
			StartedAt:      &now,
		}

		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("failed to create loop run: %w", err)
		}

		result = &loop.TriggerRunAtomicResult{Run: run, Loop: &l}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleConcurrencyPolicy handles concurrency policy when max concurrent runs is reached.
func (r *loopRunRepo) handleConcurrencyPolicy(tx *gorm.DB, l *loop.Loop, params *loop.TriggerRunAtomicParams, result **loop.TriggerRunAtomicResult) error {
	var maxNumber int
	tx.Model(&loop.LoopRun{}).
		Where("loop_id = ?", l.ID).
		Select("COALESCE(MAX(run_number), 0)").
		Scan(&maxNumber)

	now := time.Now()
	skippedRun := &loop.LoopRun{
		OrganizationID: l.OrganizationID,
		LoopID:         l.ID,
		RunNumber:      maxNumber + 1,
		Status:         loop.RunStatusSkipped,
		TriggerType:    params.TriggerType,
		TriggerSource:  &params.TriggerSource,
		FinishedAt:     &now,
	}
	if err := tx.Create(skippedRun).Error; err != nil {
		return err
	}

	reason := "max concurrent runs reached"
	switch l.ConcurrencyPolicy {
	case loop.ConcurrencyPolicyQueue:
		reason = "queued (not yet implemented)"
	case loop.ConcurrencyPolicyReplace:
		reason = "replace (not yet implemented)"
	}

	*result = &loop.TriggerRunAtomicResult{
		Run:     skippedRun,
		Loop:    l,
		Skipped: true,
		Reason:  reason,
	}
	return nil
}

// Compile-time interface compliance check
var _ loop.LoopRunRepository = (*loopRunRepo)(nil)
