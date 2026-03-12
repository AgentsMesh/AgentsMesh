package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runnerlog"
	"gorm.io/gorm"
)

// Sentinel errors for runner log repository operations.
var (
	ErrRunnerLogNotFound = errors.New("runner log record not found")
	ErrStaleStatus       = errors.New("record already in terminal status")
)

// Compile-time interface check.
var _ runnerlog.Repository = (*runnerLogRepository)(nil)

type runnerLogRepository struct{ db *gorm.DB }

// NewRunnerLogRepository creates a new RunnerLog repository backed by GORM.
func NewRunnerLogRepository(db *gorm.DB) runnerlog.Repository {
	return &runnerLogRepository{db: db}
}

func (r *runnerLogRepository) Create(ctx context.Context, log *runnerlog.RunnerLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *runnerLogRepository) GetByRequestID(ctx context.Context, requestID string) (*runnerlog.RunnerLog, error) {
	var out runnerlog.RunnerLog
	if err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&out).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *runnerLogRepository) UpdateStatus(ctx context.Context, requestID string, runnerID int64, status string, sizeBytes int64, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	// Only update size_bytes when non-zero to avoid overwriting previous values
	if sizeBytes > 0 {
		updates["size_bytes"] = sizeBytes
	}
	// Only update error_message when non-empty
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	if runnerlog.IsTerminalStatus(status) {
		now := time.Now()
		updates["completed_at"] = &now
	}

	// State machine: reject updates to already-terminal records
	result := r.db.WithContext(ctx).Model(&runnerlog.RunnerLog{}).
		Where("request_id = ? AND runner_id = ? AND status NOT IN ?", requestID, runnerID, []string{runnerlog.StatusCompleted, runnerlog.StatusFailed}).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Distinguish between not-found and already-terminal
		existing, err := r.GetByRequestID(ctx, requestID)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrRunnerLogNotFound
		}
		return ErrStaleStatus
	}
	return nil
}

func (r *runnerLogRepository) MarkFailed(ctx context.Context, requestID string, errorMessage string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&runnerlog.RunnerLog{}).
		Where("request_id = ? AND status NOT IN ?", requestID, []string{runnerlog.StatusCompleted, runnerlog.StatusFailed}).
		Updates(map[string]interface{}{
			"status":        runnerlog.StatusFailed,
			"error_message": errorMessage,
			"completed_at":  &now,
		})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

const maxListLimit = 100

func (r *runnerLogRepository) ListByRunner(ctx context.Context, orgID, runnerID int64, limit, offset int) ([]*runnerlog.RunnerLog, error) {
	var logs []*runnerlog.RunnerLog
	if limit <= 0 {
		limit = 20
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND runner_id = ?", orgID, runnerID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&logs).Error
	return logs, err
}
