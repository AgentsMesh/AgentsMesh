package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestPodCoordinatorUpdateActivity(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a pod
	initialTime := time.Now().Add(-1 * time.Hour)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, last_activity) VALUES (?, ?, ?, ?)`,
		"test-pod-1", 1, "running", initialTime)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Update activity
	err := pc.UpdateActivity(ctx, "test-pod-1")
	if err != nil {
		t.Fatalf("UpdateActivity error: %v", err)
	}

	// Verify last_activity was updated
	var lastActivity time.Time
	db.Raw(`SELECT last_activity FROM pods WHERE pod_key = ?`, "test-pod-1").Scan(&lastActivity)

	if lastActivity.Before(initialTime.Add(30 * time.Minute)) {
		t.Error("last_activity should have been updated to recent time")
	}
}

func TestPodCoordinatorMarkDisconnected(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a running pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"test-pod-2", 1, agentpod.StatusRunning)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Mark disconnected
	err := pc.MarkDisconnected(ctx, "test-pod-2")
	if err != nil {
		t.Fatalf("MarkDisconnected error: %v", err)
	}

	// Verify status was updated
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "test-pod-2").Scan(&status)

	if status != agentpod.StatusDisconnected {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusDisconnected)
	}
}

func TestPodCoordinatorMarkDisconnectedOnlyRunning(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a completed pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"test-pod-3", 1, agentpod.StatusCompleted)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Mark disconnected should not affect completed pod
	err := pc.MarkDisconnected(ctx, "test-pod-3")
	if err != nil {
		t.Fatalf("MarkDisconnected error: %v", err)
	}

	// Verify status was NOT changed
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "test-pod-3").Scan(&status)

	if status != agentpod.StatusCompleted {
		t.Errorf("completed pod status should not change: got %q", status)
	}
}

func TestPodCoordinatorMarkReconnected(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a disconnected pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"test-pod-4", 1, agentpod.StatusDisconnected)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Mark reconnected
	err := pc.MarkReconnected(ctx, "test-pod-4")
	if err != nil {
		t.Fatalf("MarkReconnected error: %v", err)
	}

	// Verify status was updated
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "test-pod-4").Scan(&status)

	if status != agentpod.StatusRunning {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusRunning)
	}
}

func TestPodCoordinatorMarkReconnectedOnlyDisconnected(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a completed pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"test-pod-5", 1, agentpod.StatusCompleted)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Mark reconnected should not affect completed pod
	err := pc.MarkReconnected(ctx, "test-pod-5")
	if err != nil {
		t.Fatalf("MarkReconnected error: %v", err)
	}

	// Verify status was NOT changed
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "test-pod-5").Scan(&status)

	if status != agentpod.StatusCompleted {
		t.Errorf("completed pod status should not change: got %q", status)
	}
}
