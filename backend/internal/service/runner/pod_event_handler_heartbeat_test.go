package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestHandleHeartbeat(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "heartbeat-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"heartbeat-pod-1", r.ID, agentpod.StatusRunning)

	// Send heartbeat (using Proto type)
	data := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{
			{PodKey: "heartbeat-pod-1", Status: "running"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify heartbeat was recorded (check buffer)
	if pc.heartbeatBatcher.BufferSize() != 1 {
		t.Errorf("heartbeat should be recorded, buffer size: %d", pc.heartbeatBatcher.BufferSize())
	}
}

func TestHandleHeartbeatSyncsAgentStatus(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "hb-agent-sync-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pod with idle agent_status
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"hb-agent-pod-1", r.ID, agentpod.StatusRunning, agentpod.AgentStatusIdle)

	// Send heartbeat with AgentStatus set to executing
	data := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{
			{PodKey: "hb-agent-pod-1", Status: "running", AgentStatus: "executing"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify agent_status was updated in DB
	var agentStatus string
	db.Raw(`SELECT agent_status FROM pods WHERE pod_key = ?`, "hb-agent-pod-1").
		Scan(&agentStatus)

	if agentStatus != agentpod.AgentStatusExecuting {
		t.Errorf("agent_status: got %q, want %q", agentStatus, agentpod.AgentStatusExecuting)
	}
}

func TestHandleHeartbeatSkipsEmptyAgentStatus(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "hb-empty-agent-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pod with executing agent_status
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status, agent_status) VALUES (?, ?, ?, ?)`,
		"hb-empty-pod-1", r.ID, agentpod.StatusRunning, agentpod.AgentStatusExecuting)

	// Send heartbeat with empty AgentStatus
	data := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{
			{PodKey: "hb-empty-pod-1", Status: "running", AgentStatus: ""},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify agent_status was NOT modified (should still be executing)
	var agentStatus string
	db.Raw(`SELECT agent_status FROM pods WHERE pod_key = ?`, "hb-empty-pod-1").
		Scan(&agentStatus)

	if agentStatus != agentpod.AgentStatusExecuting {
		t.Errorf("agent_status should not be modified when heartbeat AgentStatus is empty: got %q, want %q",
			agentStatus, agentpod.AgentStatusExecuting)
	}
}

func TestHandleHeartbeatReconcilePods(t *testing.T) {
	pc, _, tr, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "reconcile-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create pods in DB
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"reconcile-pod-1", r.ID, agentpod.StatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"reconcile-pod-2", r.ID, agentpod.StatusRunning)

	// Send heartbeat with only pod-1 (using Proto type)
	data := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{
			{PodKey: "reconcile-pod-1", Status: "running"},
		},
	}

	// Need orphanMissThreshold heartbeats for pod-2 to become orphaned
	for i := 0; i < orphanMissThreshold; i++ {
		pc.handleHeartbeat(r.ID, data)
	}

	// Verify pod-1 is still running and registered
	var status1 string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "reconcile-pod-1").Scan(&status1)
	if status1 != agentpod.StatusRunning {
		t.Errorf("pod-1 status: got %q, want %q", status1, agentpod.StatusRunning)
	}
	if !tr.IsPodRegistered("reconcile-pod-1") {
		t.Error("pod-1 should be registered with terminal router")
	}

	// Verify pod-2 is orphaned
	var status2 string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "reconcile-pod-2").Scan(&status2)
	if status2 != agentpod.StatusOrphaned {
		t.Errorf("pod-2 status: got %q, want %q", status2, agentpod.StatusOrphaned)
	}
}

func TestHandleHeartbeatRestoreOrphanedPod(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "restore-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create an orphaned pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"orphan-pod-1", r.ID, agentpod.StatusOrphaned)

	// Send heartbeat reporting the orphaned pod as running (using Proto type)
	data := &runnerv1.HeartbeatData{
		Pods: []*runnerv1.PodInfo{
			{PodKey: "orphan-pod-1", Status: "running"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify pod was restored
	var status string
	db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "orphan-pod-1").Scan(&status)
	if status != agentpod.StatusRunning {
		t.Errorf("orphaned pod should be restored: got %q, want %q", status, agentpod.StatusRunning)
	}
}

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
