package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func TestReconcilePods(t *testing.T) {
	pc, _, tr, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "reconcile-test-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create multiple pods
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-1", r.ID, agentpod.StatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-2", r.ID, agentpod.StatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-3", r.ID, agentpod.StatusInitializing)

	ctx := context.Background()
	reportedPods := map[string]bool{
		"recon-pod-1": true,
		// pod-2 and pod-3 are NOT reported
	}

	// Need orphanMissThreshold reconcile calls for unreported pods to become orphaned
	for i := 0; i < orphanMissThreshold; i++ {
		pc.reconcilePods(ctx, r.ID, reportedPods)
	}

	// Verify pod-1 is registered
	if !tr.IsPodRegistered("recon-pod-1") {
		t.Error("pod-1 should be registered")
	}

	// Verify pod-2 and pod-3 are orphaned
	var status2, status3 string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "recon-pod-2").Scan(&status2)
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "recon-pod-3").Scan(&status3)

	if status2 != agentpod.StatusOrphaned {
		t.Errorf("pod-2 should be orphaned: got %q", status2)
	}
	if status3 != agentpod.StatusOrphaned {
		t.Errorf("pod-3 should be orphaned: got %q", status3)
	}
}

func TestReconcilePodsCompletedNotAffected(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "completed-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a completed pod (should not be affected by reconciliation)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"completed-pod-1", r.ID, agentpod.StatusCompleted)

	ctx := context.Background()
	reportedPods := map[string]bool{} // Empty - no pods reported

	pc.reconcilePods(ctx, r.ID, reportedPods)

	// Verify completed pod is NOT changed
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "completed-pod-1").Scan(&status)
	if status != agentpod.StatusCompleted {
		t.Errorf("completed pod should not be affected: got %q", status)
	}
}

func TestReconcilePodsOrphanedCallsStatusChangeCallback(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "orphan-callback-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create running pods
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"orphan-cb-pod-1", r.ID, agentpod.StatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"orphan-cb-pod-2", r.ID, agentpod.StatusRunning)

	// Track callback invocations
	var callbackCalls []struct {
		podKey      string
		status      string
		agentStatus string
	}
	pc.SetStatusChangeCallback(func(podKey, status, agentStatus string) {
		callbackCalls = append(callbackCalls, struct {
			podKey      string
			status      string
			agentStatus string
		}{podKey, status, agentStatus})
	})

	ctx := context.Background()
	// Report empty pods - both should become orphaned after threshold
	reportedPods := map[string]bool{}

	// Need orphanMissThreshold reconcile calls for unreported pods to become orphaned
	for i := 0; i < orphanMissThreshold; i++ {
		pc.reconcilePods(ctx, r.ID, reportedPods)
	}

	// Verify both pods are orphaned in DB
	var status1, status2 string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "orphan-cb-pod-1").Scan(&status1)
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "orphan-cb-pod-2").Scan(&status2)
	if status1 != agentpod.StatusOrphaned {
		t.Errorf("pod-1 should be orphaned: got %q", status1)
	}
	if status2 != agentpod.StatusOrphaned {
		t.Errorf("pod-2 should be orphaned: got %q", status2)
	}

	// Verify callback was called for each orphaned pod
	if len(callbackCalls) != 2 {
		t.Errorf("expected 2 callback calls, got %d", len(callbackCalls))
	}

	// Check each callback invocation
	orphanedPods := make(map[string]bool)
	for _, call := range callbackCalls {
		if call.status != agentpod.StatusOrphaned {
			t.Errorf("callback status should be %q, got %q", agentpod.StatusOrphaned, call.status)
		}
		if call.agentStatus != "" {
			t.Errorf("callback agentStatus should be empty, got %q", call.agentStatus)
		}
		orphanedPods[call.podKey] = true
	}

	if !orphanedPods["orphan-cb-pod-1"] {
		t.Error("callback should have been called for orphan-cb-pod-1")
	}
	if !orphanedPods["orphan-cb-pod-2"] {
		t.Error("callback should have been called for orphan-cb-pod-2")
	}
}

func TestReconcilePodsRestoredCallsStatusChangeCallback(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "restore-callback-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create orphaned pods
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"restore-cb-pod-1", r.ID, agentpod.StatusOrphaned)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"restore-cb-pod-2", r.ID, agentpod.StatusOrphaned)

	// Track callback invocations
	var callbackCalls []struct {
		podKey      string
		status      string
		agentStatus string
	}
	pc.SetStatusChangeCallback(func(podKey, status, agentStatus string) {
		callbackCalls = append(callbackCalls, struct {
			podKey      string
			status      string
			agentStatus string
		}{podKey, status, agentStatus})
	})

	ctx := context.Background()
	// Report both orphaned pods as running
	reportedPods := map[string]bool{
		"restore-cb-pod-1": true,
		"restore-cb-pod-2": true,
	}

	pc.reconcilePods(ctx, r.ID, reportedPods)

	// Verify both pods are restored to running in DB
	var status1, status2 string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "restore-cb-pod-1").Scan(&status1)
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "restore-cb-pod-2").Scan(&status2)
	if status1 != agentpod.StatusRunning {
		t.Errorf("pod-1 should be running: got %q", status1)
	}
	if status2 != agentpod.StatusRunning {
		t.Errorf("pod-2 should be running: got %q", status2)
	}

	// Verify callback was called for each restored pod
	if len(callbackCalls) != 2 {
		t.Errorf("expected 2 callback calls, got %d", len(callbackCalls))
	}

	// Check each callback invocation
	restoredPods := make(map[string]bool)
	for _, call := range callbackCalls {
		if call.status != agentpod.StatusRunning {
			t.Errorf("callback status should be %q, got %q", agentpod.StatusRunning, call.status)
		}
		if call.agentStatus != "" {
			t.Errorf("callback agentStatus should be empty, got %q", call.agentStatus)
		}
		restoredPods[call.podKey] = true
	}

	if !restoredPods["restore-cb-pod-1"] {
		t.Error("callback should have been called for restore-cb-pod-1")
	}
	if !restoredPods["restore-cb-pod-2"] {
		t.Error("callback should have been called for restore-cb-pod-2")
	}
}
