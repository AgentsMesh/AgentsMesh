package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func TestNewPodCoordinator(t *testing.T) {
	logger := newTestLogger()
	_, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)

	if pc == nil {
		t.Fatal("NewPodCoordinator returned nil")
	}
	if pc.podRepo != podRepo {
		t.Error("podRepo not set correctly")
	}
	if pc.runnerRepo != runnerRepo {
		t.Error("runnerRepo not set correctly")
	}
	if pc.connectionManager != cm {
		t.Error("connectionManager not set correctly")
	}
	if pc.podRouter != tr {
		t.Error("podRouter not set correctly")
	}
	if pc.heartbeatBatcher != hb {
		t.Error("heartbeatBatcher not set correctly")
	}
}

func TestPodCoordinatorSetStatusChangeCallback(t *testing.T) {
	logger := newTestLogger()
	_, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)

	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		// Callback set for testing
	})

	if pc.onStatusChange == nil {
		t.Error("onStatusChange should be set")
	}
}

func TestPodCoordinatorIncrementPods(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         "online",
		CurrentPods:    0,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Increment pods
	err := pc.IncrementPods(ctx, r.ID)
	if err != nil {
		t.Fatalf("IncrementPods error: %v", err)
	}

	// Verify
	var updated runner.Runner
	if err := db.First(&updated, r.ID).Error; err != nil {
		t.Fatalf("failed to get runner: %v", err)
	}
	if updated.CurrentPods != 1 {
		t.Errorf("CurrentPods: got %d, want 1", updated.CurrentPods)
	}

	// Increment again
	err = pc.IncrementPods(ctx, r.ID)
	if err != nil {
		t.Fatalf("IncrementPods error: %v", err)
	}

	if err := db.First(&updated, r.ID).Error; err != nil {
		t.Fatalf("failed to get runner: %v", err)
	}
	if updated.CurrentPods != 2 {
		t.Errorf("CurrentPods: got %d, want 2", updated.CurrentPods)
	}
}
