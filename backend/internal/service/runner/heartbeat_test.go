package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Heartbeat Tests ---

func TestHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create runner directly
	r := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "test-runner",
		Description:       "Test",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r)

	// Send heartbeat
	err := service.Heartbeat(ctx, r.ID, 2)
	if err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	// Check runner status was updated
	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected Status '%s', got %s", runner.RunnerStatusOnline, updated.Status)
	}
	if updated.CurrentPods != 2 {
		t.Errorf("expected CurrentPods 2, got %d", updated.CurrentPods)
	}
	if updated.LastHeartbeat == nil {
		t.Error("expected LastHeartbeat to be set")
	}
}

func TestUpdateHeartbeatWithPods(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create runner directly
	r := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "test-runner",
		Description:       "Test",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r)

	pods := []HeartbeatPodInfo{
		{PodKey: "pod-1", Status: "running"},
		{PodKey: "pod-2", Status: "running"},
	}

	err := service.UpdateHeartbeatWithPods(ctx, r.ID, pods, "1.0.0")
	if err != nil {
		t.Fatalf("failed to update heartbeat with pods: %v", err)
	}

	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.CurrentPods != 2 {
		t.Errorf("expected 2 pods, got %d", updated.CurrentPods)
	}
}

func TestUpdateHeartbeatWithPodsNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	err := service.UpdateHeartbeatWithPods(ctx, 99999, nil, "1.0.0")
	if err != ErrRunnerNotFound {
		t.Errorf("expected ErrRunnerNotFound, got %v", err)
	}
}

func TestMarkOfflineRunners(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create runner directly
	r := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "test-runner",
		Description:       "Test",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 5,
		IsEnabled:         true,
	}
	db.Create(r)

	// Mark as online with a recent heartbeat
	now := time.Now()
	db.Model(&runner.Runner{}).Where("id = ?", r.ID).Updates(map[string]interface{}{
		"status":         runner.RunnerStatusOnline,
		"last_heartbeat": now,
	})

	// Mark offline with a timeout longer than since heartbeat
	service.MarkOfflineRunners(ctx, time.Hour)

	// Should still be online
	updated, _ := service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected status online, got %s", updated.Status)
	}

	// Set old heartbeat
	oldTime := now.Add(-2 * time.Hour)
	db.Model(&runner.Runner{}).Where("id = ?", r.ID).Update("last_heartbeat", oldTime)

	// Now should be marked offline
	service.MarkOfflineRunners(ctx, time.Hour)

	updated, _ = service.GetRunner(ctx, r.ID)
	if updated.Status != runner.RunnerStatusOffline {
		t.Errorf("expected status offline, got %s", updated.Status)
	}
}
