package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/tasks"
	"gorm.io/gorm"
)

var _ tasks.TaskExecutionRepository = (*taskExecutionRepo)(nil)

type taskExecutionRepo struct{ db *gorm.DB }

// NewTaskExecutionRepository creates a new TaskExecutionRepository backed by GORM.
func NewTaskExecutionRepository(db *gorm.DB) tasks.TaskExecutionRepository {
	return &taskExecutionRepo{db: db}
}

func (r *taskExecutionRepo) UpdateStatus(ctx context.Context, taskID int64, status string, errorMsg string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	if status == tasks.TaskStatusSuccess || status == tasks.TaskStatusFailed {
		now := time.Now()
		updates["finished_at"] = &now
	}

	return r.db.WithContext(ctx).
		Model(&tasks.TaskExecution{}).
		Where("id = ?", taskID).
		Updates(updates).Error
}

func (r *taskExecutionRepo) GetByID(ctx context.Context, taskID int64) (*tasks.TaskExecution, error) {
	var task tasks.TaskExecution
	err := r.db.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}
