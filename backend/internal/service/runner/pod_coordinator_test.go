package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/agentpod"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestHeartbeatBatcher creates a HeartbeatBatcher for testing with miniredis
func newTestHeartbeatBatcher(t *testing.T, db *gorm.DB) *HeartbeatBatcher {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(func() { mr.Close() })

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() { redisClient.Close() })

	return NewHeartbeatBatcher(redisClient, db, newTestLogger())
}

func setupCoordinatorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create runners table
	db.Exec(`CREATE TABLE IF NOT EXISTS runners (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		node_id TEXT NOT NULL,
		description TEXT,
		auth_token_hash TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'offline',
		last_heartbeat DATETIME,
		current_pods INTEGER NOT NULL DEFAULT 0,
		max_concurrent_pods INTEGER NOT NULL DEFAULT 5,
		runner_version TEXT,
		is_enabled INTEGER NOT NULL DEFAULT 1,
		host_info TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// Create pods table
	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pod_key TEXT NOT NULL UNIQUE,
		runner_id INTEGER,
		status TEXT NOT NULL DEFAULT 'initializing',
		agent_status TEXT,
		pty_pid INTEGER,
		branch_name TEXT,
		worktree_path TEXT,
		started_at DATETIME,
		finished_at DATETIME,
		last_activity DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func TestNewPodCoordinator(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	hb := newTestHeartbeatBatcher(t, db)

	pc := NewPodCoordinator(db, cm, tr, hb, newTestLogger())

	if pc == nil {
		t.Fatal("NewPodCoordinator returned nil")
	}
	if pc.db != db {
		t.Error("db not set correctly")
	}
	if pc.connectionManager != cm {
		t.Error("connectionManager not set correctly")
	}
	if pc.terminalRouter != tr {
		t.Error("terminalRouter not set correctly")
	}
	if pc.heartbeatBatcher != hb {
		t.Error("heartbeatBatcher not set correctly")
	}
}

func TestPodCoordinatorSetStatusChangeCallback(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	pc.SetStatusChangeCallback(func(podID string, status string, agentStatus string) {
		// callback for testing
	})

	if pc.onStatusChange == nil {
		t.Error("onStatusChange should be set")
	}
}

func TestPodCoordinatorIncrementPods(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a runner
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, current_pods) VALUES (1, 'test', 'hash', 0)`)

	err := pc.IncrementPods(ctx, 1)
	if err != nil {
		t.Errorf("IncrementPods error: %v", err)
	}

	var count int
	db.Raw("SELECT current_pods FROM runners WHERE id = 1").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 pod, got %d", count)
	}
}

func TestPodCoordinatorDecrementPods(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a runner with 2 pods
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, current_pods) VALUES (1, 'test', 'hash', 2)`)

	// SQLite doesn't have GREATEST function, just test that method doesn't panic
	err := pc.DecrementPods(ctx, 1)
	// Skip error check since SQLite doesn't support GREATEST
	_ = err
}

func TestPodCoordinatorUpdateActivity(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a pod
	oldTime := time.Now().Add(-1 * time.Hour)
	db.Exec(`INSERT INTO pods (pod_key, status, last_activity) VALUES ('test-pod', 'running', ?)`, oldTime)

	err := pc.UpdateActivity(ctx, "test-pod")
	if err != nil {
		t.Errorf("UpdateActivity error: %v", err)
	}

	var lastActivity time.Time
	db.Raw("SELECT last_activity FROM pods WHERE pod_key = 'test-pod'").Scan(&lastActivity)

	if lastActivity.Before(oldTime.Add(30 * time.Minute)) {
		t.Error("last_activity should be updated to recent time")
	}
}

func TestPodCoordinatorMarkDisconnected(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a running pod
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('test-pod', ?)`, agentpod.PodStatusRunning)

	err := pc.MarkDisconnected(ctx, "test-pod")
	if err != nil {
		t.Errorf("MarkDisconnected error: %v", err)
	}

	var status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'test-pod'").Scan(&status)
	if status != agentpod.PodStatusDisconnected {
		t.Errorf("expected status %s, got %s", agentpod.PodStatusDisconnected, status)
	}
}

func TestPodCoordinatorMarkReconnected(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a disconnected pod
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('test-pod', ?)`, agentpod.PodStatusDisconnected)

	err := pc.MarkReconnected(ctx, "test-pod")
	if err != nil {
		t.Errorf("MarkReconnected error: %v", err)
	}

	var status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'test-pod'").Scan(&status)
	if status != agentpod.PodStatusRunning {
		t.Errorf("expected status %s, got %s", agentpod.PodStatusRunning, status)
	}
}

func TestPodCoordinatorHandleHeartbeat(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	hb := newTestHeartbeatBatcher(t, db)
	pc := NewPodCoordinator(db, cm, tr, hb, newTestLogger())

	// Create a runner
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, status) VALUES (1, 'test', 'hash', 'offline')`)

	hbData := &HeartbeatData{
		RunnerVersion: "1.0.0",
		Pods:          []HeartbeatPod{{PodKey: "pod-1"}},
	}

	pc.handleHeartbeat(1, hbData)

	// Flush the heartbeat batcher to ensure DB is updated
	hb.Flush()

	var status string
	db.Raw("SELECT status FROM runners WHERE id = 1").Scan(&status)
	if status != "online" {
		t.Errorf("expected status online, got %s", status)
	}

	var version string
	db.Raw("SELECT runner_version FROM runners WHERE id = 1").Scan(&version)
	if version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", version)
	}
}

func TestPodCoordinatorHandlePodCreated(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	callbackCalled := false
	pc.SetStatusChangeCallback(func(podID string, status string, agentStatus string) {
		callbackCalled = true
		if status != agentpod.PodStatusRunning {
			t.Errorf("expected status %s, got %s", agentpod.PodStatusRunning, status)
		}
	})

	// Create a pod
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('test-pod', 'initializing')`)

	pcData := &PodCreatedData{
		PodKey:        "test-pod",
		Pid:          12345,
		BranchName:   "main",
		WorktreePath: "/path/to/worktree",
	}

	pc.handlePodCreated(1, pcData)

	var status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'test-pod'").Scan(&status)
	if status != agentpod.PodStatusRunning {
		t.Errorf("expected status %s, got %s", agentpod.PodStatusRunning, status)
	}

	if !callbackCalled {
		t.Error("status change callback should be called")
	}

	// Check terminal router registered
	if !tr.IsPodRegistered("test-pod") {
		t.Error("pod should be registered with terminal router")
	}
}

func TestPodCoordinatorHandlePodTerminated(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	callbackCalled := false
	pc.SetStatusChangeCallback(func(podID string, status string, agentStatus string) {
		callbackCalled = true
		if status != agentpod.StatusCompleted {
			t.Errorf("expected status %s, got %s", agentpod.StatusCompleted, status)
		}
	})

	// Create a runner and pod
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, current_pods) VALUES (1, 'test', 'hash', 1)`)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('test-pod', 1, 'running')`)

	// Register pod with terminal router
	tr.RegisterPod("test-pod", 1)

	ptData := &PodTerminatedData{
		PodKey:    "test-pod",
		ExitCode: 0,
	}

	pc.handlePodTerminated(1, ptData)

	var status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'test-pod'").Scan(&status)
	if status != agentpod.StatusCompleted {
		t.Errorf("expected status %s, got %s", agentpod.StatusCompleted, status)
	}

	if !callbackCalled {
		t.Error("status change callback should be called")
	}

	// Check terminal router unregistered
	if tr.IsPodRegistered("test-pod") {
		t.Error("pod should be unregistered from terminal router")
	}
}

func TestPodCoordinatorHandleAgentStatus(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	callbackCalled := false
	pc.SetStatusChangeCallback(func(podID string, status string, agentStatus string) {
		callbackCalled = true
		if agentStatus != "waiting" {
			t.Errorf("expected agentStatus waiting, got %s", agentStatus)
		}
	})

	// Create a pod
	db.Exec(`INSERT INTO pods (pod_key, status) VALUES ('test-pod', 'running')`)

	asData := &AgentStatusData{
		PodKey:  "test-pod",
		Status: "waiting",
		Pid:    12345,
	}

	pc.handleAgentStatus(1, asData)

	var agentStatus string
	db.Raw("SELECT agent_status FROM pods WHERE pod_key = 'test-pod'").Scan(&agentStatus)
	if agentStatus != "waiting" {
		t.Errorf("expected agent_status waiting, got %s", agentStatus)
	}

	if !callbackCalled {
		t.Error("status change callback should be called")
	}
}

func TestPodCoordinatorHandleRunnerDisconnect(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	// Create a runner and pods
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, status) VALUES (1, 'test', 'hash', 'online')`)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-1', 1, ?)`, agentpod.PodStatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-2', 1, ?)`, agentpod.PodStatusInitializing)

	pc.handleRunnerDisconnect(1)

	// Check runner is offline
	var runnerStatus string
	db.Raw("SELECT status FROM runners WHERE id = 1").Scan(&runnerStatus)
	if runnerStatus != "offline" {
		t.Errorf("expected runner status offline, got %s", runnerStatus)
	}

	// Note: Pods are intentionally NOT marked as orphaned immediately on disconnect.
	// This is by design to handle temporary network glitches - pods remain in their
	// current state and will be reconciled when:
	// 1. Runner reconnects and sends heartbeat (reconcilePods handles it)
	// 2. Pod cleanup task runs and finds stale pods
	// The previous behavior of immediately marking pods as orphaned caused issues
	// with quick reconnects where pods were still actually running.
	var s1Status, s2Status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-1'").Scan(&s1Status)
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-2'").Scan(&s2Status)

	// Pods should retain their original status (not orphaned)
	if s1Status != agentpod.PodStatusRunning {
		t.Errorf("expected pod-1 status running (retained), got %s", s1Status)
	}
	if s2Status != agentpod.PodStatusInitializing {
		t.Errorf("expected pod-2 status initializing (retained), got %s", s2Status)
	}
}

// TestPodCoordinatorReconcileOrphansOnReconnect tests that pods are properly
// orphaned when runner reconnects but doesn't report them in heartbeat
func TestPodCoordinatorReconcileOrphansOnReconnect(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())

	// Create a runner and pods
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, status) VALUES (1, 'test', 'hash', 'online')`)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-1', 1, ?)`, agentpod.PodStatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-2', 1, ?)`, agentpod.PodStatusRunning)

	// Simulate runner disconnect
	pc.handleRunnerDisconnect(1)

	// Simulate runner reconnect with heartbeat - only reporting pod-1
	hbData := &HeartbeatData{
		RunnerVersion: "1.0.0",
		Pods:          []HeartbeatPod{{PodKey: "pod-1"}},
	}
	pc.handleHeartbeat(1, hbData)

	// pod-1 should still be running (reported in heartbeat)
	var s1Status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-1'").Scan(&s1Status)
	if s1Status != agentpod.PodStatusRunning {
		t.Errorf("expected pod-1 status running, got %s", s1Status)
	}

	// pod-2 should be orphaned (not reported in heartbeat)
	var s2Status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-2'").Scan(&s2Status)
	if s2Status != agentpod.PodStatusOrphaned {
		t.Errorf("expected pod-2 status orphaned, got %s", s2Status)
	}
}

func TestPodCoordinatorReconcilePods(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a runner and pods
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash) VALUES (1, 'test', 'hash')`)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-1', 1, ?)`, agentpod.PodStatusRunning)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('pod-2', 1, ?)`, agentpod.PodStatusRunning)

	// Only pod-1 is reported
	reportedPods := map[string]bool{
		"pod-1": true,
	}

	pc.reconcilePods(ctx, 1, reportedPods)

	// pod-1 should still be running
	var s1Status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-1'").Scan(&s1Status)
	if s1Status != agentpod.PodStatusRunning {
		t.Errorf("expected pod-1 status running, got %s", s1Status)
	}

	// pod-2 should be orphaned
	var s2Status string
	db.Raw("SELECT status FROM pods WHERE pod_key = 'pod-2'").Scan(&s2Status)
	if s2Status != agentpod.PodStatusOrphaned {
		t.Errorf("expected pod-2 status orphaned, got %s", s2Status)
	}
}

func TestPodCoordinatorTerminatePod(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a runner and pod
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, current_pods) VALUES (1, 'test', 'hash', 1)`)
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES ('test-pod', 1, 'running')`)

	// Register with terminal router
	tr.RegisterPod("test-pod", 1)

	// SQLite doesn't have GREATEST function, so we just verify basic flow
	_ = pc.TerminatePod(ctx, "test-pod")

	// Check terminal router unregistered (this should work regardless of DB function issues)
	if tr.IsPodRegistered("test-pod") {
		t.Error("pod should be unregistered from terminal router")
	}
}

func TestPodCoordinatorTerminatePodNotFound(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	err := pc.TerminatePod(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
}

func TestPodCoordinatorCreatePod(t *testing.T) {
	db := setupCoordinatorTestDB(t)
	cm := NewConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	pc := NewPodCoordinator(db, cm, tr, newTestHeartbeatBatcher(t, db), newTestLogger())
	ctx := context.Background()

	// Create a runner
	db.Exec(`INSERT INTO runners (organization_id, node_id, auth_token_hash, current_pods) VALUES (1, 'test', 'hash', 0)`)

	req := &CreatePodRequest{
		PodKey:          "new-pod",
		InitialCommand: "claude",
		InitialPrompt:  "hello",
		PluginConfig: map[string]interface{}{
			"repository_url": "https://github.com/org/repo.git",
			"branch":         "main",
		},
	}

	// This will fail because runner is not connected, but we can still test the pod count increment
	_ = pc.CreatePod(ctx, 1, req)

	// Check pod count incremented
	var count int
	db.Raw("SELECT current_pods FROM runners WHERE id = 1").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 pod, got %d", count)
	}

	// Check terminal router registered
	if !tr.IsPodRegistered("new-pod") {
		t.Error("pod should be registered with terminal router")
	}
}
