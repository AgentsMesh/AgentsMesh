package infra

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// loopRepo implements loop.LoopRepository using GORM
type loopRepo struct {
	db *gorm.DB
}

// NewLoopRepository creates a new loop repository
func NewLoopRepository(db *gorm.DB) loop.LoopRepository {
	return &loopRepo{db: db}
}

func (r *loopRepo) Create(ctx context.Context, l *loop.Loop) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *loopRepo) GetByID(ctx context.Context, id int64) (*loop.Loop, error) {
	var l loop.Loop
	if err := r.db.WithContext(ctx).First(&l, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (r *loopRepo) GetBySlug(ctx context.Context, orgID int64, slug string) (*loop.Loop, error) {
	var l loop.Loop
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND slug = ?", orgID, slug).
		First(&l).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (r *loopRepo) List(ctx context.Context, filter *loop.ListFilter) ([]*loop.Loop, int64, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ?", filter.OrganizationID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		query = query.Where("status != ?", loop.StatusArchived)
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
		escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(filter.Query)
		q := "%" + escaped + "%"
		query = query.Where("name ILIKE ? OR slug ILIKE ? OR description ILIKE ?", q, q, q)
	}

	var total int64
	if err := query.Model(&loop.Loop{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit == 0 {
		limit = 20
	}

	var loops []*loop.Loop
	if err := query.Order("created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&loops).Error; err != nil {
		return nil, 0, err
	}

	return loops, total, nil
}

func (r *loopRepo) Update(ctx context.Context, id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return r.db.WithContext(ctx).
		Model(&loop.Loop{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete atomically deletes a loop and its associated loop_runs, but only if
// the loop has no active (pending/running) runs.
func (r *loopRepo) Delete(ctx context.Context, orgID int64, slug string) (int64, error) {
	var affected int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var l loop.Loop
		result := tx.
			Where("organization_id = ? AND slug = ?", orgID, slug).
			Where("NOT EXISTS (SELECT 1 FROM loop_runs lr WHERE lr.loop_id = loops.id AND lr.status IN (?, ?) AND lr.finished_at IS NULL)",
				loop.RunStatusPending, loop.RunStatusRunning).
			First(&l)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				var count int64
				tx.Model(&loop.Loop{}).
					Where("organization_id = ? AND slug = ?", orgID, slug).
					Count(&count)
				if count > 0 {
					return loop.ErrHasActiveRuns
				}
				return nil
			}
			return result.Error
		}

		if err := tx.Where("loop_id = ?", l.ID).Delete(&loop.LoopRun{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&l).Error; err != nil {
			return err
		}
		affected = 1
		return nil
	})
	return affected, err
}

func (r *loopRepo) GetDueCronLoops(ctx context.Context, orgIDs []int64) ([]*loop.Loop, error) {
	var loops []*loop.Loop
	query := r.db.WithContext(ctx).
		Where("status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at <= ?",
			loop.StatusEnabled, time.Now())
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&loops).Error
	return loops, err
}

// ClaimCronLoop atomically claims a cron loop with SKIP LOCKED and advances next_run_at.
func (r *loopRepo) ClaimCronLoop(ctx context.Context, loopID int64, nextRunAt *time.Time) (bool, error) {
	claimed := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var l loop.Loop
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("id = ? AND status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at <= ?",
				loopID, loop.StatusEnabled, time.Now()).
			First(&l).Error
		if err != nil {
			return nil
		}

		if nextRunAt != nil {
			tx.Model(&l).Update("next_run_at", nextRunAt)
		} else {
			fallback := time.Now().Add(1 * time.Hour)
			tx.Model(&l).Update("next_run_at", fallback)
		}

		claimed = true
		return nil
	})

	return claimed, err
}

// FindLoopsNeedingNextRun returns enabled cron loops with next_run_at IS NULL.
func (r *loopRepo) FindLoopsNeedingNextRun(ctx context.Context, orgIDs []int64) ([]*loop.Loop, error) {
	var loops []*loop.Loop
	query := r.db.WithContext(ctx).
		Where("status = ? AND cron_expression IS NOT NULL AND cron_expression != '' AND next_run_at IS NULL",
			loop.StatusEnabled)
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&loops).Error
	return loops, err
}

// IncrementRunStats atomically increments run statistics counters.
func (r *loopRepo) IncrementRunStats(ctx context.Context, loopID int64, status string, lastRunAt time.Time) error {
	updates := map[string]interface{}{
		"total_runs":  gorm.Expr("total_runs + 1"),
		"last_run_at": lastRunAt,
		"updated_at":  time.Now(),
	}

	switch status {
	case loop.RunStatusCompleted:
		updates["successful_runs"] = gorm.Expr("successful_runs + 1")
	case loop.RunStatusFailed, loop.RunStatusTimeout, loop.RunStatusCancelled:
		updates["failed_runs"] = gorm.Expr("failed_runs + 1")
	}

	return r.db.WithContext(ctx).
		Model(&loop.Loop{}).
		Where("id = ?", loopID).
		Updates(updates).Error
}

// Compile-time interface compliance check
var _ loop.LoopRepository = (*loopRepo)(nil)
