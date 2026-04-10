package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestPodCoordinatorCreatePod(t *testing.T) {
	// Note: This test verifies the CreatePod flow when a proper command sender is available.
	// We use a mock command sender to test the coordinator logic.
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a runner and add connection
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         "online",
		CurrentPods:    0,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Add a mock gRPC connection and mark it as initialized
	stream := newMockRunnerStreamWithTesting(t)
	rc := cm.AddConnection(r.ID, "test-node", "test-org", stream)
	rc.SetInitialized(true, []string{"claude"})

	// Create coordinator and set mock command sender that succeeds
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)
	ctx := context.Background()

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "new-pod-1",
		LaunchCommand: "claude",
	}

	err := pc.CreatePod(ctx, r.ID, cmd)
	if err != nil {
		t.Fatalf("CreatePod error: %v", err)
	}

	// Verify pod count was incremented
	var updated runner.Runner
	if err := db.First(&updated, r.ID).Error; err != nil {
		t.Fatalf("failed to get runner: %v", err)
	}
	if updated.CurrentPods != 1 {
		t.Errorf("CurrentPods: got %d, want 1", updated.CurrentPods)
	}

	// Note: Pod is NOT registered with terminal router at this point.
	// Registration happens when Runner confirms creation via handlePodCreated.
	// This is by design - we don't want stale routes if pod creation fails.
	if tr.IsPodRegistered("new-pod-1") {
		t.Error("pod should NOT be registered yet (registration happens on PodCreated event)")
	}
}

func TestPodCoordinatorCreatePodWithoutCommandSender(t *testing.T) {
	// Test that CreatePod returns error when commandSender is not set
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

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

	// Create coordinator WITHOUT setting command sender
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
	}

	err := pc.CreatePod(ctx, r.ID, cmd)
	if err != ErrCommandSenderNotSet {
		t.Errorf("CreatePod should return ErrCommandSenderNotSet, got: %v", err)
	}
}

func TestPodCoordinatorTerminatePod(t *testing.T) {
	// Note: TerminatePod internally calls DecrementPods which uses GREATEST
	// SQLite doesn't support GREATEST, so this test only verifies key functionality
	// The actual decrement functionality works in PostgreSQL
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "test-node",
		Status:         "online",
		CurrentPods:    1,
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pod
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"terminate-pod-1", r.ID, agentpod.StatusRunning)

	// Register pod with terminal router
	tr.RegisterPod("terminate-pod-1", r.ID)

	// Add mock gRPC connection and mark it as initialized
	stream := newMockRunnerStreamWithTesting(t)
	rc := cm.AddConnection(r.ID, "test-node", "test-org", stream)
	rc.SetInitialized(true, []string{"claude"})

	// Create coordinator with mock command sender
	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)
	ctx := context.Background()

	// TerminatePod will fail due to GREATEST on SQLite, but we verify
	// the pod is unregistered from terminal router before the DB update
	_ = pc.TerminatePod(ctx, "terminate-pod-1")

	// Verify pod was unregistered from terminal router (happens before DB update)
	if tr.IsPodRegistered("terminate-pod-1") {
		t.Error("pod should be unregistered from terminal router")
	}

	// Verify terminate was called on mock
	if mockSender.TerminatePodCalls != 1 {
		t.Errorf("TerminatePodCalls: got %d, want 1", mockSender.TerminatePodCalls)
	}
}

func TestPodCoordinatorTerminatePodNotFound(t *testing.T) {
	logger := newTestLogger()
	_, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Try to terminate non-existent pod
	err := pc.TerminatePod(ctx, "non-existent-pod")
	if err == nil {
		t.Error("TerminatePod should return error for non-existent pod")
	}
}

func TestPodCoordinatorTerminatePodAlreadyTerminated(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	r := &runner.Runner{OrganizationID: 1, NodeID: "test-node", Status: "online", CurrentPods: 0}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	for _, status := range []string{agentpod.StatusCompleted, agentpod.StatusTerminated, agentpod.StatusError, agentpod.StatusOrphaned} {
		db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
			"pod-"+status, r.ID, status)

		err := pc.TerminatePod(context.Background(), "pod-"+status)
		if err != ErrPodAlreadyTerminated {
			t.Errorf("status=%s: expected ErrPodAlreadyTerminated, got: %v", status, err)
		}
	}
	if mockSender.TerminatePodCalls != 0 {
		t.Errorf("should not send gRPC for terminal pods, got %d calls", mockSender.TerminatePodCalls)
	}
}

func TestPodCoordinatorTerminatePodFiresSSECallback(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	r := &runner.Runner{OrganizationID: 1, NodeID: "test-node", Status: "online", CurrentPods: 1}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"sse-pod", r.ID, agentpod.StatusRunning)

	stream := newMockRunnerStreamWithTesting(t)
	rc := cm.AddConnection(r.ID, "test-node", "test-org", stream)
	rc.SetInitialized(true, []string{"claude"})

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	var callbackPodKey, callbackStatus string
	pc.SetStatusChangeCallback(func(podKey, status, agentStatus string) {
		callbackPodKey = podKey
		callbackStatus = status
	})

	// TerminatePod will fail on DecrementPods (GREATEST on SQLite) but SSE fires before that
	_ = pc.TerminatePod(context.Background(), "sse-pod")

	if callbackPodKey != "sse-pod" {
		t.Errorf("SSE callback podKey: got %q, want %q", callbackPodKey, "sse-pod")
	}
	if callbackStatus != agentpod.StatusCompleted {
		t.Errorf("SSE callback status: got %q, want %q", callbackStatus, agentpod.StatusCompleted)
	}
}

func TestPodCoordinatorTerminatePodGRPCFailStillUpdatesDB(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	r := &runner.Runner{OrganizationID: 1, NodeID: "test-node", Status: "online", CurrentPods: 1}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"grpc-fail-pod", r.ID, agentpod.StatusRunning)

	stream := newMockRunnerStreamWithTesting(t)
	rc := cm.AddConnection(r.ID, "test-node", "test-org", stream)
	rc.SetInitialized(true, []string{"claude"})

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{
		TerminatePodErr: fmt.Errorf("runner disconnected"),
	}
	pc.SetCommandSender(mockSender)

	_ = pc.TerminatePod(context.Background(), "grpc-fail-pod")

	// gRPC failed but pod should still be updated
	var status string
	db.Raw("SELECT status FROM pods WHERE pod_key = ?", "grpc-fail-pod").Scan(&status)
	if status != agentpod.StatusCompleted {
		t.Errorf("pod status: got %q, want %q", status, agentpod.StatusCompleted)
	}
}

func TestPodCoordinatorTerminatePodConcurrentRace(t *testing.T) {
	logger := newTestLogger()
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)

	r := &runner.Runner{OrganizationID: 1, NodeID: "test-node", Status: "online", CurrentPods: 1}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}
	db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"race-pod", r.ID, agentpod.StatusRunning)

	stream := newMockRunnerStreamWithTesting(t)
	rc := cm.AddConnection(r.ID, "test-node", "test-org", stream)
	rc.SetInitialized(true, []string{"claude"})

	pc := NewPodCoordinator(podStore, runnerRepo, cm, tr, hb, logger)
	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	// Simulate concurrent terminate by pre-setting pod to completed
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", agentpod.StatusCompleted, "race-pod")

	// TerminatePod should detect via UpdateByKeyAndActiveStatus rowsAffected=0
	err := pc.TerminatePod(context.Background(), "race-pod")
	if err != ErrPodAlreadyTerminated {
		t.Errorf("expected ErrPodAlreadyTerminated for TOCTOU race, got: %v", err)
	}
}
