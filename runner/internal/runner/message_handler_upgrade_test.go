package runner

import (
	"context"
	"fmt"
	"os"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// testReleaseDetector implements updater.ReleaseDetector for testing.
type testReleaseDetector struct {
	detectLatestFn  func(ctx context.Context) (*updater.ReleaseInfo, bool, error)
	detectVersionFn func(ctx context.Context, version string) (*updater.ReleaseInfo, bool, error)
	updateBinaryFn  func(ctx context.Context, release *updater.ReleaseInfo, execPath string) error
}

func (d *testReleaseDetector) DetectLatest(ctx context.Context) (*updater.ReleaseInfo, bool, error) {
	if d.detectLatestFn != nil {
		return d.detectLatestFn(ctx)
	}
	return nil, false, fmt.Errorf("not configured")
}

func (d *testReleaseDetector) DetectVersion(ctx context.Context, version string) (*updater.ReleaseInfo, bool, error) {
	if d.detectVersionFn != nil {
		return d.detectVersionFn(ctx, version)
	}
	return nil, false, fmt.Errorf("not configured")
}

func (d *testReleaseDetector) UpdateBinary(ctx context.Context, release *updater.ReleaseInfo, execPath string) error {
	if d.updateBinaryFn != nil {
		return d.updateBinaryFn(ctx, release, execPath)
	}
	return fmt.Errorf("not configured")
}

func newTestRunnerForUpgrade(podCount int) *Runner {
	store := NewInMemoryPodStore()
	for i := 0; i < podCount; i++ {
		store.Put(fmt.Sprintf("pod-%d", i), &Pod{
			PodKey: fmt.Sprintf("pod-%d", i),
			Status: PodStatusRunning,
		})
	}

	r := &Runner{
		cfg:          &config.Config{},
		podStore:     store,
		runCtx:       context.Background(),
		upgradeCoord: newUpgradeController(store.Count),
	}
	return r
}

func getUpgradeStatuses(mockConn *client.MockConnection) []*runnerv1.UpgradeStatusEvent {
	events := mockConn.GetEvents()
	var statuses []*runnerv1.UpgradeStatusEvent
	for _, e := range events {
		if e.Type == "upgrade_status" {
			if evt, ok := e.Data.(*runnerv1.UpgradeStatusEvent); ok {
				statuses = append(statuses, evt)
			}
		}
	}
	return statuses
}

// noUpdateDetector simulates "already up to date"
func noUpdateDetector() *testReleaseDetector {
	return &testReleaseDetector{
		detectLatestFn: func(ctx context.Context) (*updater.ReleaseInfo, bool, error) {
			return nil, false, nil // no update available
		},
	}
}

// failDetector simulates a network/API error during update check
func failDetector() *testReleaseDetector {
	return &testReleaseDetector{
		detectLatestFn: func(ctx context.Context) (*updater.ReleaseInfo, bool, error) {
			return nil, false, fmt.Errorf("network error")
		},
		detectVersionFn: func(ctx context.Context, version string) (*updater.ReleaseInfo, bool, error) {
			return nil, false, fmt.Errorf("network error")
		},
	}
}

// successDetector simulates a successful update flow:
// DetectLatest finds v2.0.0, UpdateBinary writes a fake binary to execPath.
func successDetector(t *testing.T) *testReleaseDetector {
	t.Helper()
	release := &updater.ReleaseInfo{
		Version:  "v2.0.0",
		AssetURL: "https://example.com/runner.tar.gz",
	}
	return &testReleaseDetector{
		detectLatestFn: func(ctx context.Context) (*updater.ReleaseInfo, bool, error) {
			return release, true, nil
		},
		detectVersionFn: func(ctx context.Context, version string) (*updater.ReleaseInfo, bool, error) {
			return release, true, nil
		},
		updateBinaryFn: func(ctx context.Context, rel *updater.ReleaseInfo, execPath string) error {
			return os.WriteFile(execPath, []byte("fake-binary"), 0o755)
		},
	}
}

func TestOnUpgradeRunner_NoUpdater(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-1",
	})
	if err == nil {
		t.Fatal("expected error when updater is nil")
	}

	statuses := getUpgradeStatuses(mockConn)
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status event, got %d", len(statuses))
	}
	if statuses[0].Phase != "failed" {
		t.Errorf("expected phase=failed, got %q", statuses[0].Phase)
	}
	if statuses[0].Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestOnUpgradeRunner_ActivePods_Rejected(t *testing.T) {
	r := newTestRunnerForUpgrade(2)
	r.SetUpdater(updater.New("1.0.0"))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-2",
		Force:     false,
	})
	if err == nil {
		t.Fatal("expected error when pods are running")
	}
	if !contains(err.Error(), "active pod") {
		t.Errorf("error should mention active pods, got: %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status event, got %d", len(statuses))
	}
	if statuses[0].Phase != "failed" {
		t.Errorf("expected phase=failed, got %q", statuses[0].Phase)
	}

	// Draining should NOT be set (rejected before entering draining)
	if r.IsDraining() {
		t.Error("should not enter draining when upgrade is rejected")
	}
}

func TestOnUpgradeRunner_ActivePods_ForceAllowed(t *testing.T) {
	r := newTestRunnerForUpgrade(1)
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-3",
		Force:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	if len(statuses) == 0 {
		t.Fatal("expected status events")
	}

	// First status should be "checking"
	if statuses[0].Phase != "checking" {
		t.Errorf("expected first phase=checking, got %q", statuses[0].Phase)
	}
}

func TestOnUpgradeRunner_UpdateCheckFails(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(failDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-4",
	})
	if err != nil {
		t.Fatalf("OnUpgradeRunner should not return error (runs upgrade async): %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "failed" {
		t.Errorf("expected last phase=failed, got %q", lastStatus.Phase)
	}

	// Draining should be restored after failure
	if r.IsDraining() {
		t.Error("draining should be false after failed upgrade")
	}
}

func TestOnUpgradeRunner_AlreadyUpToDate(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	r.SetUpdater(updater.New("1.0.0", updater.WithReleaseDetector(noUpdateDetector())))
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	err := handler.OnUpgradeRunner(&runnerv1.UpgradeRunnerCommand{
		RequestId: "req-5",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statuses := getUpgradeStatuses(mockConn)
	lastStatus := statuses[len(statuses)-1]
	if lastStatus.Phase != "completed" {
		t.Errorf("expected last phase=completed, got %q", lastStatus.Phase)
	}
	if lastStatus.Progress != 100 {
		t.Errorf("expected progress=100, got %d", lastStatus.Progress)
	}

	// Draining should be restored
	if r.IsDraining() {
		t.Error("draining should be false after already-up-to-date")
	}
}
