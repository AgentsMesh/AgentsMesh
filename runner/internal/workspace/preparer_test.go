package workspace

import (
	"context"
	"os"
	"testing"
	"time"
)

// --- Test Preparer ---

func TestNewPreparer(t *testing.T) {
	step := NewScriptPreparationStep("echo hello", time.Minute)
	preparer := NewPreparer(step)

	if preparer == nil {
		t.Fatal("NewPreparer returned nil")
		return
	}

	if preparer.StepCount() != 1 {
		t.Errorf("StepCount: got %v, want 1", preparer.StepCount())
	}
}

func TestNewPreparerFromScript(t *testing.T) {
	preparer := NewPreparerFromScript("echo hello", 300)

	if preparer == nil {
		t.Fatal("NewPreparerFromScript returned nil")
		return
	}

	if preparer.StepCount() != 1 {
		t.Errorf("StepCount: got %v, want 1", preparer.StepCount())
	}
}

func TestNewPreparerFromScriptEmpty(t *testing.T) {
	preparer := NewPreparerFromScript("", 300)

	if preparer != nil {
		t.Error("NewPreparerFromScript should return nil for empty script")
	}
}

func TestNewPreparerFromScriptDefaultTimeout(t *testing.T) {
	preparer := NewPreparerFromScript("echo hello", 0)

	if preparer == nil {
		t.Fatal("NewPreparerFromScript returned nil")
	}
}

func TestPreparerAddStep(t *testing.T) {
	preparer := NewPreparer()

	if preparer.StepCount() != 0 {
		t.Errorf("initial StepCount: got %v, want 0", preparer.StepCount())
	}

	step := NewScriptPreparationStep("echo hello", time.Minute)
	preparer.AddStep(step)

	if preparer.StepCount() != 1 {
		t.Errorf("StepCount after add: got %v, want 1", preparer.StepCount())
	}
}

func TestPreparerPrepareEmpty(t *testing.T) {
	preparer := NewPreparer()
	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: t.TempDir(),
	}

	err := preparer.Prepare(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreparerPrepareSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	step := NewScriptPreparationStep("echo hello", time.Minute)
	preparer := NewPreparer(step)

	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: tmpDir,
	}

	err := preparer.Prepare(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreparerPrepareFailure(t *testing.T) {
	tmpDir := t.TempDir()
	step := NewScriptPreparationStep("exit 1", time.Minute)
	preparer := NewPreparer(step)

	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: tmpDir,
	}

	err := preparer.Prepare(context.Background(), ctx)
	if err == nil {
		t.Error("expected error for failed script")
	}

	if _, ok := err.(*PreparationError); !ok {
		t.Error("error should be a PreparationError")
	}
}

// mockPreparationStep for testing
type mockPreparationStep struct {
	name      string
	execError error
	executed  bool
}

func (m *mockPreparationStep) Name() string {
	return m.name
}

func (m *mockPreparationStep) Execute(ctx context.Context, prepCtx *PreparationContext) error {
	m.executed = true
	return m.execError
}

func TestPreparerStopsOnError(t *testing.T) {
	step1 := &mockPreparationStep{name: "step1"}
	step2 := &mockPreparationStep{name: "step2", execError: os.ErrNotExist}
	step3 := &mockPreparationStep{name: "step3"}

	preparer := NewPreparer(step1, step2, step3)
	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: t.TempDir(),
	}

	err := preparer.Prepare(context.Background(), ctx)
	if err == nil {
		t.Error("expected error")
	}

	if !step1.executed {
		t.Error("step1 should be executed")
	}

	if !step2.executed {
		t.Error("step2 should be executed")
	}

	if step3.executed {
		t.Error("step3 should not be executed after error")
	}
}

func TestPreparerMultipleSteps(t *testing.T) {
	tmpDir := t.TempDir()

	step1 := NewScriptPreparationStep("echo step1", time.Minute)
	step2 := NewScriptPreparationStep("echo step2", time.Minute)

	preparer := NewPreparer(step1, step2)

	if preparer.StepCount() != 2 {
		t.Errorf("StepCount: got %v, want 2", preparer.StepCount())
	}

	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: tmpDir,
	}

	err := preparer.Prepare(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreparerContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	step := NewScriptPreparationStep("sleep 5", time.Minute)
	preparer := NewPreparer(step)

	prepCtx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: tmpDir,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := preparer.Prepare(ctx, prepCtx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}