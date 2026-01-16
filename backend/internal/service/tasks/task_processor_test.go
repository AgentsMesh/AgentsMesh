package tasks

import (
	"context"
	"testing"
	"time"

	infraTasks "github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
)

func TestTaskExecutionModel(t *testing.T) {
	te := TaskExecution{
		ID:       1,
		TaskType: "mr_sync",
		Status:   TaskStatusPending,
	}

	if te.TableName() != "task_executions" {
		t.Errorf("TableName() = %s, want task_executions", te.TableName())
	}
	if te.TaskType != "mr_sync" {
		t.Errorf("TaskType = %s, want mr_sync", te.TaskType)
	}
}

func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"Pending", TaskStatusPending, "pending"},
		{"Running", TaskStatusRunning, "running"},
		{"Processing", TaskStatusProcessing, "processing"},
		{"Success", TaskStatusSuccess, "success"},
		{"Failed", TaskStatusFailed, "failed"},
		{"Canceled", TaskStatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestProcessResult(t *testing.T) {
	result := &ProcessResult{
		ProcessedCount: 5,
		Processed: []ProcessedTask{
			{TaskID: 1, TaskType: "sync", Status: "success", Success: true},
			{TaskID: 2, TaskType: "sync", Status: "failed", Success: false},
		},
		Errors: []ProcessError{
			{TaskID: 3, TaskType: "sync", Error: "timeout"},
		},
	}

	if result.ProcessedCount != 5 {
		t.Errorf("ProcessedCount = %d, want 5", result.ProcessedCount)
	}
	if len(result.Processed) != 2 {
		t.Errorf("len(Processed) = %d, want 2", len(result.Processed))
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}

func TestProcessedTask(t *testing.T) {
	task := ProcessedTask{
		TaskID:   100,
		TaskType: "pipeline_check",
		Status:   "success",
		Success:  true,
	}

	if task.TaskID != 100 {
		t.Errorf("TaskID = %d, want 100", task.TaskID)
	}
	if !task.Success {
		t.Error("expected Success to be true")
	}
}

func TestProcessError(t *testing.T) {
	err := ProcessError{
		TaskID:   200,
		TaskType: "mr_sync",
		Error:    "connection refused",
	}

	if err.TaskID != 200 {
		t.Errorf("TaskID = %d, want 200", err.TaskID)
	}
	if err.Error != "connection refused" {
		t.Errorf("Error = %s, want 'connection refused'", err.Error)
	}
}

func TestTaskExecutionFullModel(t *testing.T) {
	now := time.Now()
	te := TaskExecution{
		ID:                1,
		TaskType:          "mr_sync",
		TaskSubtype:       "gitlab",
		Status:            TaskStatusRunning,
		GitLabProjectID:   "12345",
		GitLabPipelineID:  67890,
		GitLabPipelineURL: "https://gitlab.com/pipeline/67890",
		TriggeredBy:       "user@example.com",
		TriggerParams:     `{"branch": "main"}`,
		ErrorMessage:      "",
		StartedAt:         &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if te.GitLabProjectID != "12345" {
		t.Errorf("GitLabProjectID = %s, want 12345", te.GitLabProjectID)
	}
	if te.GitLabPipelineID != 67890 {
		t.Errorf("GitLabPipelineID = %d, want 67890", te.GitLabPipelineID)
	}
	if te.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestNewTaskProcessorService(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	processor := NewTaskProcessorService(db, redisClient, logger)

	if processor == nil {
		t.Fatal("expected non-nil processor")
	}
	if processor.db != db {
		t.Error("expected processor.db to be set")
	}
	if processor.handlers == nil {
		t.Error("expected handlers map to be initialized")
	}
}

// MockTaskHandler implements TaskHandler for testing
type MockTaskHandler struct {
	taskType           string
	processedPipelines []*infraTasks.WatchedPipeline
	failedPipelines    []*infraTasks.WatchedPipeline
	completionError    error
	failureError       error
}

func (m *MockTaskHandler) GetTaskType() string {
	return m.taskType
}

func (m *MockTaskHandler) ProcessCompletion(ctx context.Context, pipeline *infraTasks.WatchedPipeline) error {
	m.processedPipelines = append(m.processedPipelines, pipeline)
	return m.completionError
}

func (m *MockTaskHandler) ProcessFailure(ctx context.Context, pipeline *infraTasks.WatchedPipeline, errorMsg string) error {
	m.failedPipelines = append(m.failedPipelines, pipeline)
	return m.failureError
}

func TestTaskProcessorService_RegisterHandler(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	processor := NewTaskProcessorService(db, redisClient, logger)

	handler := &MockTaskHandler{taskType: "test_task"}
	processor.RegisterHandler(handler)

	types := processor.GetRegisteredTypes()
	if len(types) != 1 {
		t.Errorf("expected 1 handler, got %d", len(types))
	}
	if types[0] != "test_task" {
		t.Errorf("expected task type 'test_task', got '%s'", types[0])
	}
}

func TestTaskProcessorService_GetRegisteredTypes(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	processor := NewTaskProcessorService(db, redisClient, logger)

	// Register multiple handlers
	processor.RegisterHandler(&MockTaskHandler{taskType: "type_a"})
	processor.RegisterHandler(&MockTaskHandler{taskType: "type_b"})
	processor.RegisterHandler(&MockTaskHandler{taskType: "type_c"})

	types := processor.GetRegisteredTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(types))
	}
}

func TestTaskProcessorService_Process_NoHandlers(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	processor := NewTaskProcessorService(db, redisClient, logger)

	result, err := processor.Process(context.Background())
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.ProcessedCount != 0 {
		t.Errorf("expected 0 processed, got %d", result.ProcessedCount)
	}
}

func TestBaseTaskHandler_UpdateTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	handler := &BaseTaskHandler{
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
	}

	// Insert a task
	db.Exec(`INSERT INTO task_executions (id, task_type, status) VALUES (1, 'test', 'pending')`)

	err := handler.UpdateTaskStatus(context.Background(), 1, TaskStatusRunning, "")
	if err != nil {
		t.Fatalf("UpdateTaskStatus() error = %v", err)
	}

	// Verify status changed
	var status string
	db.Raw("SELECT status FROM task_executions WHERE id = 1").Scan(&status)
	if status != TaskStatusRunning {
		t.Errorf("expected status 'running', got '%s'", status)
	}
}

func TestBaseTaskHandler_UpdateTaskStatus_WithError(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	handler := &BaseTaskHandler{
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
	}

	// Insert a task
	db.Exec(`INSERT INTO task_executions (id, task_type, status) VALUES (2, 'test', 'running')`)

	err := handler.UpdateTaskStatus(context.Background(), 2, TaskStatusFailed, "Connection timeout")
	if err != nil {
		t.Fatalf("UpdateTaskStatus() error = %v", err)
	}

	// Verify status and error message
	var status, errorMsg string
	db.Raw("SELECT status, error_message FROM task_executions WHERE id = 2").Row().Scan(&status, &errorMsg)
	if status != TaskStatusFailed {
		t.Errorf("expected status 'failed', got '%s'", status)
	}
	if errorMsg != "Connection timeout" {
		t.Errorf("expected error message 'Connection timeout', got '%s'", errorMsg)
	}
}

func TestBaseTaskHandler_GetTaskExecution(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	handler := &BaseTaskHandler{
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
	}

	// Insert a task using GORM to ensure correct column mapping
	te := TaskExecution{
		ID:              3,
		TaskType:        "sync",
		TaskSubtype:     "gitlab",
		Status:          "pending",
		GitLabProjectID: "12345",
	}
	db.Create(&te)

	task, err := handler.GetTaskExecution(context.Background(), 3)
	if err != nil {
		t.Fatalf("GetTaskExecution() error = %v", err)
	}
	if task.ID != 3 {
		t.Errorf("expected ID 3, got %d", task.ID)
	}
	if task.TaskType != "sync" {
		t.Errorf("expected TaskType 'sync', got '%s'", task.TaskType)
	}
	if task.GitLabProjectID != "12345" {
		t.Errorf("expected GitLabProjectID '12345', got '%s'", task.GitLabProjectID)
	}
}

func TestBaseTaskHandler_GetTaskExecution_NotFound(t *testing.T) {
	db := setupTestDB(t)
	_, redisClient := setupTestRedis(t)
	logger := testLogger()

	handler := &BaseTaskHandler{
		DB:     db,
		Redis:  redisClient,
		Logger: logger,
	}

	_, err := handler.GetTaskExecution(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent task")
	}
}
