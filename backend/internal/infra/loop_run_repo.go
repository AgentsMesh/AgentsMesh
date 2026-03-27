package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"gorm.io/gorm"
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
		if isNotFound(err) {
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
		if isNotFound(err) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
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

// Compile-time interface compliance check
var _ loop.LoopRunRepository = (*loopRunRepo)(nil)
