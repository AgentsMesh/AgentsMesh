package webhooks

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	return testkit.SetupTestDB(t)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ===========================================
// WebhookContext Tests
// ===========================================

func TestNewWebhookContext(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	payload := map[string]interface{}{
		"object_kind": "pipeline",
		"project": map[string]interface{}{
			"id": float64(123),
		},
		"object_attributes": map[string]interface{}{
			"id":     float64(456),
			"status": "success",
			"iid":    float64(789),
		},
	}

	wc := NewWebhookContext(ctx, db, payload)

	if wc.ObjectKind != "pipeline" {
		t.Errorf("expected ObjectKind 'pipeline', got %s", wc.ObjectKind)
	}
	if wc.PipelineID != 456 {
		t.Errorf("expected PipelineID 456, got %d", wc.PipelineID)
	}
	if wc.PipelineStatus != "success" {
		t.Errorf("expected PipelineStatus 'success', got %s", wc.PipelineStatus)
	}
	if wc.MRIID != 789 {
		t.Errorf("expected MRIID 789, got %d", wc.MRIID)
	}
}

func TestNewWebhookContextEmptyPayload(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	payload := map[string]interface{}{}

	wc := NewWebhookContext(ctx, db, payload)

	if wc.ObjectKind != "" {
		t.Errorf("expected empty ObjectKind, got %s", wc.ObjectKind)
	}
	if wc.ProjectID != "" {
		t.Errorf("expected empty ProjectID, got %s", wc.ProjectID)
	}
}

func TestWebhookContextResults(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	wc := NewWebhookContext(ctx, db, map[string]interface{}{})

	// Add result
	wc.AddResult("handler1", map[string]string{"status": "ok"})

	// Get result
	result, ok := wc.GetResult("handler1")
	if !ok {
		t.Error("expected to find result for handler1")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}

	// Get non-existent result
	_, ok = wc.GetResult("handler2")
	if ok {
		t.Error("expected not to find result for handler2")
	}
}

// ===========================================
// HandlerRegistry Tests
// ===========================================

type mockHandler struct {
	canHandle bool
	handleErr error
	result    map[string]interface{}
}

func (h *mockHandler) CanHandle(ctx *WebhookContext) bool {
	return h.canHandle
}

func (h *mockHandler) Handle(ctx *WebhookContext) (map[string]interface{}, error) {
	if h.handleErr != nil {
		return nil, h.handleErr
	}
	return h.result, nil
}

func TestHandlerRegistry(t *testing.T) {
	logger := testLogger()
	registry := NewHandlerRegistry(logger)

	// Register a handler
	handler := &mockHandler{canHandle: true, result: map[string]interface{}{"status": "ok"}}
	registry.Register("test_event", handler)

	// Get handler
	h, ok := registry.GetHandler("test_event")
	if !ok {
		t.Error("expected to find handler for test_event")
	}
	if h == nil {
		t.Error("expected non-nil handler")
	}

	// Get non-existent handler
	_, ok = registry.GetHandler("nonexistent")
	if ok {
		t.Error("expected not to find handler for nonexistent")
	}
}

func TestHandlerRegistryProcess(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()
	registry := NewHandlerRegistry(logger)

	// Register handler
	handler := &mockHandler{canHandle: true, result: map[string]interface{}{"status": "ok"}}
	registry.Register("pipeline", handler)

	// Create context with matching event type
	wc := NewWebhookContext(ctx, db, map[string]interface{}{"object_kind": "pipeline"})

	result, err := registry.Process(wc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
}

func TestHandlerRegistryProcessNoHandler(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()
	registry := NewHandlerRegistry(logger)

	// Create context with unknown event type
	wc := NewWebhookContext(ctx, db, map[string]interface{}{"object_kind": "unknown"})

	result, err := registry.Process(wc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "skipped" {
		t.Errorf("expected status 'skipped', got %v", result["status"])
	}
	if result["reason"] != "no_handler" {
		t.Errorf("expected reason 'no_handler', got %v", result["reason"])
	}
}

func TestHandlerRegistryProcessHandlerDeclined(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()
	registry := NewHandlerRegistry(logger)

	// Register handler that declines
	handler := &mockHandler{canHandle: false}
	registry.Register("pipeline", handler)

	wc := NewWebhookContext(ctx, db, map[string]interface{}{"object_kind": "pipeline"})

	result, err := registry.Process(wc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "skipped" {
		t.Errorf("expected status 'skipped', got %v", result["status"])
	}
	if result["reason"] != "handler_declined" {
		t.Errorf("expected reason 'handler_declined', got %v", result["reason"])
	}
}

// ===========================================
// CompositeHandler Tests
// ===========================================

func TestCompositeHandler(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()

	composite := NewCompositeHandler(logger)

	// Add sub-handlers
	handler1 := &mockHandler{canHandle: true, result: map[string]interface{}{"handler": "1"}}
	composite.AddSubHandler(handler1)

	// Check CanHandle always returns true
	wc := NewWebhookContext(ctx, db, map[string]interface{}{})
	if !composite.CanHandle(wc) {
		t.Error("CompositeHandler should always return true for CanHandle")
	}

	// Handle
	result, err := composite.Handle(wc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	subResults, ok := result["sub_results"].(map[string]interface{})
	if !ok {
		t.Fatal("expected sub_results to be a map")
	}
	// Note: getHandlerName returns the same name "handler" for all handlers,
	// so multiple handlers will overwrite each other in the results map
	if len(subResults) != 1 {
		t.Errorf("expected 1 sub result, got %d", len(subResults))
	}
}

func TestCompositeHandlerWithError(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()

	composite := NewCompositeHandler(logger)

	// Add handler that errors
	handler := &mockHandler{canHandle: true, handleErr: context.Canceled}
	composite.AddSubHandler(handler)

	wc := NewWebhookContext(ctx, db, map[string]interface{}{})
	result, err := composite.Handle(wc)

	// Should not return error (errors are captured in results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	subResults := result["sub_results"].(map[string]interface{})
	for _, subResult := range subResults {
		sr := subResult.(map[string]interface{})
		if sr["status"] != "error" {
			t.Errorf("expected sub result status 'error', got %v", sr["status"])
		}
	}
}

func TestCompositeHandlerSkipsDeclined(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := testLogger()

	composite := NewCompositeHandler(logger)

	// Add handler that declines and one that accepts
	handler1 := &mockHandler{canHandle: false}
	handler2 := &mockHandler{canHandle: true, result: map[string]interface{}{"handler": "2"}}
	composite.AddSubHandler(handler1)
	composite.AddSubHandler(handler2)

	wc := NewWebhookContext(ctx, db, map[string]interface{}{})
	result, err := composite.Handle(wc)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	subResults := result["sub_results"].(map[string]interface{})
	// Only handler2 should be in results
	if len(subResults) != 1 {
		t.Errorf("expected 1 sub result, got %d", len(subResults))
	}
}

// ===========================================
// Utility Function Tests
// ===========================================

func TestFormatID(t *testing.T) {
	// formatID converts int64 to rune string - this is a simple test
	result := formatID(65)
	if result != "A" { // 65 is 'A' in ASCII
		t.Errorf("expected 'A', got %s", result)
	}
}

func TestGetHandlerName(t *testing.T) {
	handler := &mockHandler{}
	name := getHandlerName(handler)
	if name != "handler" {
		t.Errorf("expected 'handler', got %s", name)
	}
}
