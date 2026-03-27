package infra

import (
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"context"
)

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
