package loop

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// repository implements LoopRepository
type repository struct {
	db *gorm.DB
}

// NewLoopRepository creates a new loop repository
func NewLoopRepository(db *gorm.DB) LoopRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, loop *Loop) error {
	return r.db.WithContext(ctx).Create(loop).Error
}

func (r *repository) GetByID(ctx context.Context, id int64) (*Loop, error) {
	var loop Loop
	if err := r.db.WithContext(ctx).First(&loop, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &loop, nil
}

func (r *repository) GetBySlug(ctx context.Context, orgID int64, slug string) (*Loop, error) {
	var loop Loop
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		First(&loop).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &loop, nil
}

func (r *repository) List(ctx context.Context, filter *ListFilter) ([]*Loop, int64, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", filter.OrganizationID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		query = query.Where("status != ?", StatusArchived)
	}
	if filter.ExecutionMode != "" {
		query = query.Where("execution_mode = ?", filter.ExecutionMode)
	}
	if filter.CronEnabled != nil {
		if *filter.CronEnabled {
			query = query.Where("cron_expression IS NOT NULL AND cron_expression != ''")
		} else {
			query = query.Where("cron_expression IS NULL OR cron_expression = ''")
		}
	}
	if filter.Query != "" {
		// Escape ILIKE wildcards to prevent user-injected patterns
		escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(filter.Query)
		q := "%" + escaped + "%"
		query = query.Where("name ILIKE ? OR slug ILIKE ? OR description ILIKE ?", q, q, q)
	}

	var total int64
	if err := query.Model(&Loop{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	var loops []*Loop
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&loops).Error; err != nil {
		return nil, 0, err
	}

	return loops, total, nil
}

func (r *repository) Update(ctx context.Context, id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&Loop{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete atomically deletes a loop and its associated loop_runs, but only if
// the loop has no active (pending/running) runs.
//
// Without FK CASCADE, application-level cleanup is required:
//  1. Check no active runs exist (atomic subquery)
//  2. Delete all terminal loop_runs for this loop
//  3. Delete the loop record
//
// All steps run in a single transaction to ensure consistency.
// Returns ErrHasActiveRuns if active runs exist.
func (r *repository) Delete(ctx context.Context, orgID int64, slug string) (int64, error) {
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: Find the loop, rejecting deletion if active runs exist
		var loop Loop
		result := tx.
			Where("organization_id = ? AND slug = ?", orgID, slug).
			Where("NOT EXISTS (SELECT 1 FROM loop_runs lr WHERE lr.loop_id = loops.id AND lr.status IN (?, ?) AND lr.finished_at IS NULL)",
				RunStatusPending, RunStatusRunning).
			First(&loop)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// Disambiguate: loop doesn't exist vs has active runs
				var count int64
				tx.Model(&Loop{}).
					Where("organization_id = ? AND slug = ?", orgID, slug).
					Count(&count)
				if count > 0 {
					return ErrHasActiveRuns
				}
				// Loop doesn't exist — return 0 affected
				return nil
			}
			return result.Error
		}

		// Step 2: Clean up all loop_runs (no FK CASCADE, application-level cleanup)
		if err := tx.Where("loop_id = ?", loop.ID).Delete(&LoopRun{}).Error; err != nil {
			return err
		}

		// Step 3: Delete the loop
		if err := tx.Delete(&loop).Error; err != nil {
			return err
		}
		affected = 1
		return nil
	})
	return affected, err
}

func (r *repository) GetDueCronLoops(ctx context.Context, orgIDs []int64) ([]*Loop, error) {
	var loops []*Loop
	query := r.db.WithContext(ctx).
		Where("status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at <= ?",
			StatusEnabled, time.Now())
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&loops).Error
	return loops, err
}

// ClaimCronLoop atomically claims a cron loop with SKIP LOCKED and advances next_run_at.
// Returns true if claimed, false if skipped or no longer due.
func (r *repository) ClaimCronLoop(ctx context.Context, loopID int64, nextRunAt *time.Time) (bool, error) {
	claimed := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var loop Loop
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("id = ? AND status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at <= ?",
				loopID, StatusEnabled, time.Now()).
			First(&loop).Error
		if err != nil {
			// Row was skipped (already locked) or not found/not due anymore
			return nil
		}

		// Advance next_run_at immediately inside the transaction.
		// Always advance to prevent infinite re-trigger — use a fallback if nextRunAt is nil.
		if nextRunAt != nil {
			tx.Model(&loop).Update("next_run_at", nextRunAt)
		} else {
			// Safety fallback: cron parse failed upstream — push 1 hour to prevent tight loop
			fallback := time.Now().Add(1 * time.Hour)
			tx.Model(&loop).Update("next_run_at", fallback)
		}

		claimed = true
		return nil
	})

	return claimed, err
}

// FindLoopsNeedingNextRun returns enabled cron loops with next_run_at IS NULL.
func (r *repository) FindLoopsNeedingNextRun(ctx context.Context, orgIDs []int64) ([]*Loop, error) {
	var loops []*Loop
	query := r.db.WithContext(ctx).
		Where("status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at IS NULL",
			StatusEnabled)
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&loops).Error
	return loops, err
}

// IncrementRunStats atomically increments run statistics counters.
func (r *repository) IncrementRunStats(ctx context.Context, loopID int64, status string, lastRunAt time.Time) error {
	updates := map[string]interface{}{
		"total_runs":  gorm.Expr("total_runs + 1"),
		"last_run_at": lastRunAt,
		"updated_at":  time.Now(),
	}

	switch status {
	case RunStatusCompleted:
		updates["successful_runs"] = gorm.Expr("successful_runs + 1")
	case RunStatusFailed, RunStatusTimeout, RunStatusCancelled:
		updates["failed_runs"] = gorm.Expr("failed_runs + 1")
	}

	return r.db.WithContext(ctx).
		Model(&Loop{}).
		Where("id = ?", loopID).
		Updates(updates).Error
}
