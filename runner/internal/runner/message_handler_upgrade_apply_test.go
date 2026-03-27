package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

func TestOnUpgradeRunner_SuccessfulUpdate_NoRestartFunc(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	// Create a fake executable path for Apply
	tmpDir := t.TempDir()
	fakeExec := filepath.Join(tmpDir, "runner")
	os.WriteFile(fakeExec, []byte("old-binary"), 0o755)

	r.SetUpdater(updater.New("1.0.0",
		updater.WithReleaseDetector(successDetector(t)),
		updater.WithExecPathFunc(func() (string, error) { return fakeExec, nil }),
	))
	r.SetRestartFunc(nil) // no restart function
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-6",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "completed" {
		t.Errorf("expected last phase=completed, got %q", lastStatus.Phase)
	}
	// Should report manual restart needed
	if lastStatus.Message == "" {
		t.Error("expected non-empty message about manual restart")
	}

	// Draining should be restored
	if r.IsDraining() {
		t.Error("draining should be false when no restart func and update completed")
	}
}

func TestOnUpgradeRunner_SuccessfulUpdate_WithRestartFunc(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	tmpDir := t.TempDir()
	fakeExec := filepath.Join(tmpDir, "runner")
	os.WriteFile(fakeExec, []byte("old-binary"), 0o755)

	r.SetUpdater(updater.New("1.0.0",
		updater.WithReleaseDetector(successDetector(t)),
		updater.WithExecPathFunc(func() (string, error) { return fakeExec, nil }),
	))

	restartCalled := false
	r.SetRestartFunc(func() (int, error) {
		restartCalled = true
		return 0, nil // Simulate successful restart
	})

	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-7",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !restartCalled {
		t.Error("expected restart function to be called")
	}

	// Check phases: checking -> downloading -> applying -> restarting
	statuses := getUpgradeStatuses(mockConn)
	phases := make([]string, len(statuses))
	for i, s := range statuses {
		phases[i] = s.Phase
	}

	expectedPhases := []string{"checking", "downloading", "applying", "restarting"}
	for _, expected := range expectedPhases {
		found := false
		for _, p := range phases {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected phase %q in status events, got phases: %v", expected, phases)
		}
	}
}

func TestOnUpgradeRunner_RestartFuncFails(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	tmpDir := t.TempDir()
	fakeExec := filepath.Join(tmpDir, "runner")
	os.WriteFile(fakeExec, []byte("old-binary"), 0o755)

	r.SetUpdater(updater.New("1.0.0",
		updater.WithReleaseDetector(successDetector(t)),
		updater.WithExecPathFunc(func() (string, error) { return fakeExec, nil }),
	))
	r.SetRestartFunc(func() (int, error) {
		return 0, fmt.Errorf("restart permission denied")
	})

	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	_ = handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-8",
	})

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "completed" {
		t.Errorf("expected phase=completed even when restart fails, got %q", lastStatus.Phase)
	}
	if !contains(lastStatus.Message, "restart failed") {
		t.Errorf("expected message about restart failure, got %q", lastStatus.Message)
	}

	// Draining should be restored
	if r.IsDraining() {
		t.Error("draining should be false after restart failure")
	}
}

func TestOnUpgradeRunner_RequestIdPropagated(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	requestID := "unique-request-id-12345"
	_ = handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: requestID,
	})

	statuses := getUpgradeStatuses(mockConn)
	for _, s := range statuses {
		if s.RequestId != requestID {
			t.Errorf("expected request_id=%q in all statuses, got %q", requestID, s.RequestId)
		}
	}
}

func TestOnUpgradeRunner_CurrentVersionReported(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	r.SetUpdater(updater.New("2.5.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	_ = handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-ver",
	})

	statuses := getUpgradeStatuses(mockConn)
	if len(statuses) == 0 {
		t.Fatal("expected status events")
	}
	for _, s := range statuses {
		// updater normalizes version to "v2.5.0"
		if s.CurrentVersion != "v2.5.0" {
			t.Errorf("expected current_version=v2.5.0, got %q", s.CurrentVersion)
		}
	}
}

func TestOnUpgradeRunner_TargetVersion(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	detector := &testReleaseDetector{
		detectVersionFn: func(ctx context.Context, version string) (*updater.ReleaseInfo, bool, error) {
			return nil, false, fmt.Errorf("version %s not found", version)
		},
		updateBinaryFn: func(ctx context.Context, release *updater.ReleaseInfo, execPath string) error {
			return fmt.Errorf("version not found")
		},
	}
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(detector)))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	_ = handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId:     "req-target",
		TargetVersion: "3.0.0",
	})

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "failed" {
		t.Errorf("expected phase=failed for missing version, got %q", lastStatus.Phase)
	}

	// Verify target_version is propagated in all status events
	for _, s := range statuses {
		if s.TargetVersion != "3.0.0" {
			t.Errorf("expected target_version=3.0.0 in all statuses, got %q", s.TargetVersion)
		}
	}
}

func TestOnUpgradeRunner_TargetVersionEmpty_WhenLatest(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(noUpdateDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	_ = handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-latest",
		// TargetVersion intentionally empty -- upgrade to latest
	})

	statuses := getUpgradeStatuses(mockConn)
	for _, s := range statuses {
		if s.TargetVersion != "" {
			t.Errorf("expected empty target_version for latest upgrade, got %q", s.TargetVersion)
		}
	}
}

func TestOnUpgradeRunner_SendError(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	// updater is nil to trigger immediate error path
	mockConn := client.NewMockConnection()
	mockConn.SendErr = fmt.Errorf("connection lost")
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-err",
	})
	// Should still return the primary error even if send fails
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOnUpgradeRunner_ConcurrentUpgrade_Discarded(t *testing.T) {
	r := newTestRunnerForUpgrade(0)

	// Simulate an upgrade already in progress by acquiring the flag
	if !r.TryStartUpgrade() {
		t.Fatal("should be able to start upgrade")
	}

	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	// Second upgrade while first is in progress should be silently discarded
	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-second",
	})
	if err != nil {
		t.Fatalf("concurrent upgrade should be silently discarded, got error: %v", err)
	}

	// No status events should be produced for the discarded command
	statuses := getUpgradeStatuses(mockConn)
	if len(statuses) != 0 {
		t.Errorf("expected no status events for discarded upgrade, got %d", len(statuses))
	}

	// Clean up
	r.FinishUpgrade()
}

func TestOnUpgradeRunner_ContextCancelled(t *testing.T) {
	r := newTestRunnerForUpgrade(0)

	// Set a cancelled context to simulate runner shutting down
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	r.runCtx = ctx

	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-ctx",
	})
	if err != nil {
		t.Fatalf("should not return error (upgrade runs async): %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "failed" {
		t.Errorf("expected phase=failed when context cancelled, got %q", lastStatus.Phase)
	}

	// Draining should be restored
	if r.IsDraining() {
		t.Error("draining should be false after context cancellation")
	}

	// Upgrade flag should be cleared
	if !r.TryStartUpgrade() {
		t.Error("upgrade flag should be cleared after context cancellation")
	}
}
