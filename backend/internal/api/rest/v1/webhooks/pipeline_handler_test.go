package webhooks

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDBForPipeline(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func testLoggerForPipeline() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ===========================================
// PipelineHandler Tests
// ===========================================

func TestNewPipelineHandler(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPipelineHandler(logger)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestPipelineHandlerCanHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPipelineHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		objectKind string
		pipelineID int64
		expected   bool
	}{
		{"pipeline with valid ID", "pipeline", 123, true},
		{"pipeline with zero ID", "pipeline", 0, false},
		{"non-pipeline event", "push", 123, false},
		{"empty object kind", "", 123, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := NewWebhookContext(ctx, db, map[string]interface{}{
				"object_kind": tt.objectKind,
				"object_attributes": map[string]interface{}{
					"id": float64(tt.pipelineID),
				},
			})
			if got := handler.CanHandle(wc); got != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPipelineHandlerHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPipelineHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "pipeline",
		"project": map[string]interface{}{
			"id": float64(123),
		},
		"object_attributes": map[string]interface{}{
			"id":     float64(456),
			"status": "success",
			"url":    "https://gitlab.com/project/pipelines/456",
		},
	}

	wc := NewWebhookContext(ctx, db, payload)
	result, err := handler.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	if result["pipeline_id"] != int64(456) {
		t.Errorf("expected pipeline_id 456, got %v", result["pipeline_id"])
	}
	if result["pipeline_status"] != "success" {
		t.Errorf("expected pipeline_status 'success', got %v", result["pipeline_status"])
	}
	if result["pipeline_url"] != "https://gitlab.com/project/pipelines/456" {
		t.Errorf("expected pipeline_url, got %v", result["pipeline_url"])
	}
}

func TestPipelineHandlerHandleWithoutURL(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPipelineHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "pipeline",
		"object_attributes": map[string]interface{}{
			"id":     float64(456),
			"status": "failed",
		},
	}

	wc := NewWebhookContext(ctx, db, payload)
	result, err := handler.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["pipeline_url"] != "" {
		t.Errorf("expected empty pipeline_url, got %v", result["pipeline_url"])
	}
}

// ===========================================
// MergeRequestHandler Tests
// ===========================================

func TestNewMergeRequestHandler(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewMergeRequestHandler(logger)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestMergeRequestHandlerCanHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewMergeRequestHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	tests := []struct {
		name         string
		objectKind   string
		sourceBranch string
		expected     bool
	}{
		{"MR with source branch", "merge_request", "feature/test", true},
		{"MR without source branch", "merge_request", "", false},
		{"non-MR event", "push", "feature/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"object_kind": tt.objectKind,
			}
			if tt.sourceBranch != "" || tt.objectKind == "merge_request" {
				payload["object_attributes"] = map[string]interface{}{
					"source_branch": tt.sourceBranch,
				}
			}
			wc := NewWebhookContext(ctx, db, payload)
			if got := handler.CanHandle(wc); got != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMergeRequestHandlerHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewMergeRequestHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "merge_request",
		"object_attributes": map[string]interface{}{
			"iid":           float64(42),
			"action":        "open",
			"source_branch": "feature/new-feature",
			"target_branch": "main",
			"title":         "Add new feature",
			"state":         "opened",
			"url":           "https://gitlab.com/project/-/merge_requests/42",
		},
	}

	wc := NewWebhookContext(ctx, db, payload)
	result, err := handler.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	if result["source_branch"] != "feature/new-feature" {
		t.Errorf("expected source_branch 'feature/new-feature', got %v", result["source_branch"])
	}
	if result["action"] != "open" {
		t.Errorf("expected action 'open', got %v", result["action"])
	}
	if result["title"] != "Add new feature" {
		t.Errorf("expected title 'Add new feature', got %v", result["title"])
	}
}

func TestMergeRequestHandlerHandleMissingAttrs(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewMergeRequestHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "merge_request",
	}

	wc := NewWebhookContext(ctx, db, payload)
	_, err := handler.Handle(wc)

	if err == nil {
		t.Error("expected error for missing object_attributes")
	}
}

// ===========================================
// PushHandler Tests
// ===========================================

func TestNewPushHandler(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPushHandler(logger)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestPushHandlerCanHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPushHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		objectKind string
		expected   bool
	}{
		{"push event", "push", true},
		{"non-push event", "pipeline", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := NewWebhookContext(ctx, db, map[string]interface{}{
				"object_kind": tt.objectKind,
			})
			if got := handler.CanHandle(wc); got != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPushHandlerHandle(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPushHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "push",
		"ref":         "refs/heads/main",
		"before":      "0000000000000000000000000000000000000000",
		"after":       "abc123def456",
		"commits": []interface{}{
			map[string]interface{}{"id": "abc123"},
			map[string]interface{}{"id": "def456"},
		},
	}

	wc := NewWebhookContext(ctx, db, payload)
	result, err := handler.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	if result["ref"] != "refs/heads/main" {
		t.Errorf("expected ref 'refs/heads/main', got %v", result["ref"])
	}
	if result["total_commits"] != 2 {
		t.Errorf("expected total_commits 2, got %v", result["total_commits"])
	}
}

func TestPushHandlerHandleEmptyCommits(t *testing.T) {
	logger := testLoggerForPipeline()
	handler := NewPushHandler(logger)
	db := setupTestDBForPipeline(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "push",
		"ref":         "refs/heads/feature",
	}

	wc := NewWebhookContext(ctx, db, payload)
	result, err := handler.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["total_commits"] != 0 {
		t.Errorf("expected total_commits 0, got %v", result["total_commits"])
	}
}

// ===========================================
// SetupDefaultHandlers Tests
// ===========================================

func TestSetupDefaultHandlers(t *testing.T) {
	logger := testLoggerForPipeline()
	registry := NewHandlerRegistry(logger)

	SetupDefaultHandlers(registry, logger)

	// Verify handlers are registered
	handlers := []string{"pipeline", "merge_request", "push"}
	for _, name := range handlers {
		_, ok := registry.GetHandler(name)
		if !ok {
			t.Errorf("expected handler for %s to be registered", name)
		}
	}
}

// ===========================================
// Pipeline Status Constants Tests
// ===========================================

func TestPipelineStatusConstants(t *testing.T) {
	// Verify constants are defined correctly
	if PipelineStatusPending != "pending" {
		t.Errorf("expected PipelineStatusPending 'pending', got %s", PipelineStatusPending)
	}
	if PipelineStatusRunning != "running" {
		t.Errorf("expected PipelineStatusRunning 'running', got %s", PipelineStatusRunning)
	}
	if PipelineStatusSuccess != "success" {
		t.Errorf("expected PipelineStatusSuccess 'success', got %s", PipelineStatusSuccess)
	}
	if PipelineStatusFailed != "failed" {
		t.Errorf("expected PipelineStatusFailed 'failed', got %s", PipelineStatusFailed)
	}
	if PipelineStatusCanceled != "canceled" {
		t.Errorf("expected PipelineStatusCanceled 'canceled', got %s", PipelineStatusCanceled)
	}
	if PipelineStatusSkipped != "skipped" {
		t.Errorf("expected PipelineStatusSkipped 'skipped', got %s", PipelineStatusSkipped)
	}
	if PipelineStatusManual != "manual" {
		t.Errorf("expected PipelineStatusManual 'manual', got %s", PipelineStatusManual)
	}
}
