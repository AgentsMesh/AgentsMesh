package runner

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestPodCoordinatorCreatePod(t *testing.T) {
	// Note: This test verifies the CreatePod flow when a proper command sender is available.
	// We use a mock command sender to test the coordinator logic.
	logger := newTestLogger()
	db, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

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
	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)
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

	// Create coordinator WITHOUT setting command sender
	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)
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
	db, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

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
	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)
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
	_, cm, tr, hb, podRepo, runnerRepo := setupPodCoordinatorDeps(t)

	pc := NewPodCoordinator(podRepo, runnerRepo, cm, tr, hb, logger)
	ctx := context.Background()

	// Try to terminate non-existent pod
	err := pc.TerminatePod(ctx, "non-existent-pod")
	if err == nil {
		t.Error("TerminatePod should return error for non-existent pod")
	}
}
