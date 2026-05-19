package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
	"github.com/redis/go-redis/v9"
)

type TaskHandler interface {
	GetTaskType() string

	ProcessCompletion(ctx context.Context, pipeline *infraTasks.WatchedPipeline) error

	ProcessFailure(ctx context.Context, pipeline *infraTasks.WatchedPipeline, errorMsg string) error
}

type TaskProcessorService struct {
	redis    *redis.Client
	watcher  *infraTasks.PipelineWatcher
	handlers map[string]TaskHandler
	logger   *slog.Logger
	mu       sync.RWMutex
}

func NewTaskProcessorService(
	redisClient *redis.Client,
	logger *slog.Logger,
) *TaskProcessorService {
	return &TaskProcessorService{
		redis:    redisClient,
		watcher:  infraTasks.NewPipelineWatcher(redisClient, logger),
		handlers: make(map[string]TaskHandler),
		logger:   logger,
	}
}

func (s *TaskProcessorService) RegisterHandler(handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[handler.GetTaskType()] = handler
	s.logger.Info("task handler registered", "type", handler.GetTaskType())
}

func (s *TaskProcessorService) GetRegisteredTypes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	types := make([]string, 0, len(s.handlers))
	for t := range s.handlers {
		types = append(types, t)
	}
	return types
}

type ProcessResult struct {
	ProcessedCount int
	Processed      []ProcessedTask
	Errors         []ProcessError
}

type ProcessedTask struct {
	TaskID   int64
	TaskType string
	Status   string
	Success  bool
}

type ProcessError struct {
	TaskID   int64
	TaskType string
	Error    string
}

func (s *TaskProcessorService) Process(ctx context.Context) (*ProcessResult, error) {
	result := &ProcessResult{
		Processed: []ProcessedTask{},
		Errors:    []ProcessError{},
	}

	s.mu.RLock()
	taskTypes := make([]string, 0, len(s.handlers))
	for t := range s.handlers {
		taskTypes = append(taskTypes, t)
	}
	s.mu.RUnlock()

	for _, taskType := range taskTypes {
		pipelines, err := s.watcher.GetCompletedPipelines(ctx, taskType)
		if err != nil {
			s.logger.Error("failed to get completed pipelines",
				"task_type", taskType,
				"error", err)
			continue
		}

		if len(pipelines) > 0 {
			s.logger.Info("found completed tasks to process",
				"task_type", taskType,
				"count", len(pipelines))
		}

		for _, pipeline := range pipelines {
			processed, procErr := s.processSingleTask(ctx, taskType, pipeline)
			if procErr != nil {
				result.Errors = append(result.Errors, ProcessError{
					TaskID:   pipeline.TaskID,
					TaskType: taskType,
					Error:    procErr.Error(),
				})
			} else {
				result.Processed = append(result.Processed, processed)
				result.ProcessedCount++
			}

			if err := s.watcher.MarkProcessed(ctx, pipeline.ProjectID, pipeline.PipelineID); err != nil {
				s.logger.Warn("failed to mark pipeline as processed",
					"project_id", pipeline.ProjectID,
					"pipeline_id", pipeline.PipelineID,
					"error", err)
			}
		}
	}

	return result, nil
}

func (s *TaskProcessorService) processSingleTask(
	ctx context.Context,
	taskType string,
	pipeline *infraTasks.WatchedPipeline,
) (ProcessedTask, error) {
	result := ProcessedTask{
		TaskID:   pipeline.TaskID,
		TaskType: taskType,
		Status:   pipeline.Status,
	}

	s.mu.RLock()
	handler, ok := s.handlers[taskType]
	s.mu.RUnlock()

	if !ok {
		return result, fmt.Errorf("no handler for task type: %s", taskType)
	}

	s.logger.Info("processing task",
		"task_id", pipeline.TaskID,
		"task_type", taskType,
		"status", pipeline.Status)

	var err error
	if pipeline.Status == "success" {
		err = handler.ProcessCompletion(ctx, pipeline)
	} else {
		errorMsg := fmt.Sprintf("Pipeline %s", pipeline.Status)
		if pipeline.WebURL != "" {
			errorMsg += fmt.Sprintf(". See %s", pipeline.WebURL)
		}
		err = handler.ProcessFailure(ctx, pipeline, errorMsg)
	}

	if err != nil {
		s.logger.Error("task processing failed",
			"task_id", pipeline.TaskID,
			"task_type", taskType,
			"error", err)
		return result, err
	}

	result.Success = true
	s.logger.Info("task processed successfully",
		"task_id", pipeline.TaskID,
		"task_type", taskType)

	return result, nil
}

type TaskExecution struct {
	ID              int64     `gorm:"primaryKey"`
	TaskType        string    `gorm:"size:50;not null;index"`
	TaskSubtype     string    `gorm:"size:50;index"`
	Status          string    `gorm:"size:50;not null;default:'pending'"`
	GitLabProjectID string    `gorm:"size:50;index"`
	GitLabPipelineID int64    `gorm:"index"`
	GitLabPipelineURL string  `gorm:"type:text"`
	TriggeredBy     string    `gorm:"size:100"`
	TriggerParams   string    `gorm:"type:jsonb"`
	ErrorMessage    string    `gorm:"type:text"`
	StartedAt       *time.Time
	FinishedAt      *time.Time
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (TaskExecution) TableName() string {
	return "task_executions"
}

const (
	TaskStatusPending    = "pending"
	TaskStatusRunning    = "running"
	TaskStatusProcessing = "processing"
	TaskStatusSuccess    = "success"
	TaskStatusFailed     = "failed"
	TaskStatusCanceled   = "canceled"
)

type TaskExecutionRepository interface {
	UpdateStatus(ctx context.Context, taskID int64, status string, errorMsg string) error
	GetByID(ctx context.Context, taskID int64) (*TaskExecution, error)
}

type BaseTaskHandler struct {
	Repo   TaskExecutionRepository
	Redis  *redis.Client
	Logger *slog.Logger
}

func (h *BaseTaskHandler) UpdateTaskStatus(ctx context.Context, taskID int64, status string, errorMsg string) error {
	return h.Repo.UpdateStatus(ctx, taskID, status, errorMsg)
}

func (h *BaseTaskHandler) GetTaskExecution(ctx context.Context, taskID int64) (*TaskExecution, error) {
	return h.Repo.GetByID(ctx, taskID)
}
