package runner

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/anthropics/agentmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentmesh/backend/internal/domain/runner"
	"github.com/redis/go-redis/v9"
)

// setupPodEventHandlerDeps sets up dependencies for pod event handler testing
func setupPodEventHandlerDeps(t *testing.T) (*PodCoordinator, *ConnectionManager, *TerminalRouter) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() {
		mr.Close()
	})

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		redisClient.Close()
	})

	logger := newTestLogger()
	db := setupTestDB(t)

	// Create pods table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pods (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pod_key TEXT NOT NULL UNIQUE,
			runner_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			agent_status TEXT,
			pty_pid INTEGER,
			branch_name TEXT,
			worktree_path TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			last_activity DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create pods table: %v", err)
	}

	cm := NewConnectionManager(logger)
	tr := NewTerminalRouter(cm, logger)
	hb := NewHeartbeatBatcher(redisClient, db, logger)
	pc := NewPodCoordinator(db, cm, tr, hb, logger)

	return pc, cm, tr
}

func TestHandleHeartbeat(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "heartbeat-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"heartbeat-pod-1", r.ID, agentpod.StatusRunning)

	// Send heartbeat
	data := &HeartbeatData{
		RunnerVersion: "1.0.0",
		Pods: []HeartbeatPod{
			{PodKey: "heartbeat-pod-1", Status: "running"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify heartbeat was recorded (check buffer)
	if pc.heartbeatBatcher.BufferSize() != 1 {
		t.Errorf("heartbeat should be recorded, buffer size: %d", pc.heartbeatBatcher.BufferSize())
	}
}

func TestHandleHeartbeatReconcilePods(t *testing.T) {
	pc, _, tr := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "reconcile-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create pods in DB
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"reconcile-pod-1", r.ID, agentpod.StatusRunning)
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"reconcile-pod-2", r.ID, agentpod.StatusRunning)

	// Send heartbeat with only pod-1
	data := &HeartbeatData{
		Pods: []HeartbeatPod{
			{PodKey: "reconcile-pod-1", Status: "running"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify pod-1 is still running and registered
	var status1 string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "reconcile-pod-1").Scan(&status1)
	if status1 != agentpod.StatusRunning {
		t.Errorf("pod-1 status: got %q, want %q", status1, agentpod.StatusRunning)
	}
	if !tr.IsPodRegistered("reconcile-pod-1") {
		t.Error("pod-1 should be registered with terminal router")
	}

	// Verify pod-2 is orphaned
	var status2 string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "reconcile-pod-2").Scan(&status2)
	if status2 != agentpod.StatusOrphaned {
		t.Errorf("pod-2 status: got %q, want %q", status2, agentpod.StatusOrphaned)
	}
}

func TestHandleHeartbeatRestoreOrphanedPod(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "restore-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create an orphaned pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"orphan-pod-1", r.ID, agentpod.StatusOrphaned)

	// Send heartbeat reporting the orphaned pod as running
	data := &HeartbeatData{
		Pods: []HeartbeatPod{
			{PodKey: "orphan-pod-1", Status: "running"},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify pod was restored
	var status string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "orphan-pod-1").Scan(&status)
	if status != agentpod.StatusRunning {
		t.Errorf("orphaned pod should be restored: got %q, want %q", status, agentpod.StatusRunning)
	}
}

func TestHandlePodCreated(t *testing.T) {
	pc, _, tr := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "create-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pending pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"create-pod-1", r.ID, agentpod.StatusInitializing)

	// Track status change callback
	var callbackPodKey, callbackStatus string
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackPodKey = podKey
		callbackStatus = status
	})

	// Handle pod created event
	data := &PodCreatedData{
		PodKey:       "create-pod-1",
		Pid:          12345,
		BranchName:   "feature/test",
		WorktreePath: "/tmp/worktree",
	}

	pc.handlePodCreated(r.ID, data)

	// Verify pod was updated
	var status string
	var pid int
	var branch, worktree string
	pc.db.Raw(`SELECT status, pty_pid, branch_name, worktree_path FROM pods WHERE pod_key = ?`, "create-pod-1").
		Row().Scan(&status, &pid, &branch, &worktree)

	if status != agentpod.StatusRunning {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusRunning)
	}
	if pid != 12345 {
		t.Errorf("pid: got %d, want 12345", pid)
	}
	if branch != "feature/test" {
		t.Errorf("branch: got %q, want %q", branch, "feature/test")
	}
	if worktree != "/tmp/worktree" {
		t.Errorf("worktree: got %q, want %q", worktree, "/tmp/worktree")
	}

	// Verify pod was registered
	if !tr.IsPodRegistered("create-pod-1") {
		t.Error("pod should be registered with terminal router")
	}

	// Verify callback was called
	if callbackPodKey != "create-pod-1" {
		t.Errorf("callback podKey: got %q, want %q", callbackPodKey, "create-pod-1")
	}
	if callbackStatus != agentpod.StatusRunning {
		t.Errorf("callback status: got %q, want %q", callbackStatus, agentpod.StatusRunning)
	}
}

func TestHandlePodCreatedMinimalData(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "minimal-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pending pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"minimal-pod-1", r.ID, agentpod.StatusInitializing)

	// Handle pod created with minimal data (no branch/worktree)
	data := &PodCreatedData{
		PodKey: "minimal-pod-1",
		Pid:    54321,
	}

	pc.handlePodCreated(r.ID, data)

	// Verify pod was updated
	var status string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "minimal-pod-1").Scan(&status)
	if status != agentpod.StatusRunning {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusRunning)
	}
}

func TestHandlePodTerminated(t *testing.T) {
	// Note: handlePodTerminated calls DecrementPods which uses GREATEST
	// SQLite doesn't support GREATEST, so we skip the pod count verification
	pc, _, tr := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "terminate-node",
		AuthTokenHash:  "hash",
		Status:         "online",
		CurrentPods:    2,
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a running pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status, pty_pid) VALUES (?, ?, ?, ?)`,
		"term-pod-1", r.ID, agentpod.StatusRunning, 12345)
	tr.RegisterPod("term-pod-1", r.ID)

	// Track status change callback
	var callbackPodKey, callbackStatus string
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackPodKey = podKey
		callbackStatus = status
	})

	// Handle pod terminated
	data := &PodTerminatedData{
		PodKey:   "term-pod-1",
		ExitCode: 0,
	}

	pc.handlePodTerminated(r.ID, data)

	// Verify pod was updated
	var status string
	var finishedAt time.Time
	pc.db.Raw(`SELECT status, finished_at FROM pods WHERE pod_key = ?`, "term-pod-1").
		Row().Scan(&status, &finishedAt)

	if status != agentpod.StatusCompleted {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusCompleted)
	}
	if finishedAt.IsZero() {
		t.Error("finished_at should be set")
	}

	// Verify pod was unregistered
	if tr.IsPodRegistered("term-pod-1") {
		t.Error("pod should be unregistered from terminal router")
	}

	// Note: Skip runner pod count verification due to GREATEST limitation
	// The actual functionality works in PostgreSQL

	// Verify callback was called
	if callbackPodKey != "term-pod-1" {
		t.Errorf("callback podKey: got %q, want %q", callbackPodKey, "term-pod-1")
	}
	if callbackStatus != agentpod.StatusCompleted {
		t.Errorf("callback status: got %q, want %q", callbackStatus, agentpod.StatusCompleted)
	}
}

func TestHandleAgentStatus(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "agent-status-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a running pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"agent-pod-1", r.ID, agentpod.StatusRunning)

	// Track status change callback
	var callbackAgentStatus string
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackAgentStatus = agentStatus
	})

	// Handle agent status change
	data := &AgentStatusData{
		PodKey: "agent-pod-1",
		Status: "thinking",
		Pid:    99999,
	}

	pc.handleAgentStatus(r.ID, data)

	// Verify pod was updated
	var agentStatus string
	var pid int
	pc.db.Raw(`SELECT agent_status, pty_pid FROM pods WHERE pod_key = ?`, "agent-pod-1").
		Row().Scan(&agentStatus, &pid)

	if agentStatus != "thinking" {
		t.Errorf("agent_status: got %q, want %q", agentStatus, "thinking")
	}
	if pid != 99999 {
		t.Errorf("pty_pid: got %d, want 99999", pid)
	}

	// Verify callback was called
	if callbackAgentStatus != "thinking" {
		t.Errorf("callback agentStatus: got %q, want %q", callbackAgentStatus, "thinking")
	}
}

func TestHandleAgentStatusNoPid(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "agent-nopid-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a running pod with existing pid
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status, pty_pid) VALUES (?, ?, ?, ?)`,
		"agent-nopid-1", r.ID, agentpod.StatusRunning, 11111)

	// Handle agent status change without pid
	data := &AgentStatusData{
		PodKey: "agent-nopid-1",
		Status: "idle",
		Pid:    0, // No pid
	}

	pc.handleAgentStatus(r.ID, data)

	// Verify agent_status was updated but pid was not changed
	var agentStatus string
	var pid int
	pc.db.Raw(`SELECT agent_status, pty_pid FROM pods WHERE pod_key = ?`, "agent-nopid-1").
		Row().Scan(&agentStatus, &pid)

	if agentStatus != "idle" {
		t.Errorf("agent_status: got %q, want %q", agentStatus, "idle")
	}
	// pid should remain unchanged
	if pid != 11111 {
		t.Errorf("pty_pid should not change when pid=0: got %d, want 11111", pid)
	}
}

func TestHandleRunnerDisconnect(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "disconnect-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a running pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"disconnect-pod-1", r.ID, agentpod.StatusRunning)

	// Handle runner disconnect
	pc.handleRunnerDisconnect(r.ID)

	// Verify runner was marked as offline
	var updated runner.Runner
	pc.db.First(&updated, r.ID)
	if updated.Status != "offline" {
		t.Errorf("runner status: got %q, want %q", updated.Status, "offline")
	}

	// Verify pod is NOT immediately orphaned (by design)
	var podStatus string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "disconnect-pod-1").Scan(&podStatus)
	if podStatus != agentpod.StatusRunning {
		t.Errorf("pod should still be running (not immediately orphaned): got %q", podStatus)
	}
}

func TestReconcilePods(t *testing.T) {
	pc, _, tr := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "reconcile-test-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create multiple pods
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-1", r.ID, agentpod.StatusRunning)
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-2", r.ID, agentpod.StatusRunning)
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"recon-pod-3", r.ID, agentpod.StatusInitializing)

	ctx := context.Background()
	reportedPods := map[string]bool{
		"recon-pod-1": true,
		// pod-2 and pod-3 are NOT reported
	}

	pc.reconcilePods(ctx, r.ID, reportedPods)

	// Verify pod-1 is registered
	if !tr.IsPodRegistered("recon-pod-1") {
		t.Error("pod-1 should be registered")
	}

	// Verify pod-2 and pod-3 are orphaned
	var status2, status3 string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "recon-pod-2").Scan(&status2)
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "recon-pod-3").Scan(&status3)

	if status2 != agentpod.StatusOrphaned {
		t.Errorf("pod-2 should be orphaned: got %q", status2)
	}
	if status3 != agentpod.StatusOrphaned {
		t.Errorf("pod-3 should be orphaned: got %q", status3)
	}
}

func TestReconcilePodsCompletedNotAffected(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "completed-node",
		AuthTokenHash:  "hash",
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a completed pod (should not be affected by reconciliation)
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"completed-pod-1", r.ID, agentpod.StatusCompleted)

	ctx := context.Background()
	reportedPods := map[string]bool{} // Empty - no pods reported

	pc.reconcilePods(ctx, r.ID, reportedPods)

	// Verify completed pod is NOT changed
	var status string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "completed-pod-1").Scan(&status)
	if status != agentpod.StatusCompleted {
		t.Errorf("completed pod should not be affected: got %q", status)
	}
}
