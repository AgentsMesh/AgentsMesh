package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestHandlePodCreated(t *testing.T) {
	pc, _, tr := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "create-node",
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

	// Handle pod created event (using Proto type with sandbox_path and branch_name)
	data := &runnerv1.PodCreatedEvent{
		PodKey:      "create-pod-1",
		Pid:         12345,
		SandboxPath: "/workspace/sandboxes/create-pod-1",
		BranchName:  "feature/test",
	}

	pc.handlePodCreated(r.ID, data)

	// Verify pod was updated including sandbox_path and branch_name
	var status string
	var pid int
	var sandboxPath, branchName *string
	pc.db.Raw(`SELECT status, pty_pid, sandbox_path, branch_name FROM pods WHERE pod_key = ?`, "create-pod-1").
		Row().Scan(&status, &pid, &sandboxPath, &branchName)

	if status != agentpod.StatusRunning {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusRunning)
	}
	if pid != 12345 {
		t.Errorf("pid: got %d, want 12345", pid)
	}
	if sandboxPath == nil || *sandboxPath != "/workspace/sandboxes/create-pod-1" {
		t.Errorf("sandbox_path: got %v, want %q", sandboxPath, "/workspace/sandboxes/create-pod-1")
	}
	if branchName == nil || *branchName != "feature/test" {
		t.Errorf("branch_name: got %v, want %q", branchName, "feature/test")
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
		Status:         "online",
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a pending pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"minimal-pod-1", r.ID, agentpod.StatusInitializing)

	// Handle pod created with minimal data (using Proto type)
	data := &runnerv1.PodCreatedEvent{
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

	// Handle pod terminated (using Proto type)
	data := &runnerv1.PodTerminatedEvent{
		PodKey:   "term-pod-1",
		ExitCode: 0,
	}

	pc.handlePodTerminated(r.ID, data)

	// Verify pod was updated
	var status string
	var agentStatus string
	var finishedAt time.Time
	pc.db.Raw(`SELECT status, agent_status, finished_at FROM pods WHERE pod_key = ?`, "term-pod-1").
		Row().Scan(&status, &agentStatus, &finishedAt)

	if status != agentpod.StatusCompleted {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusCompleted)
	}
	if agentStatus != agentpod.AgentStatusIdle {
		t.Errorf("agent_status: got %q, want %q", agentStatus, agentpod.AgentStatusIdle)
	}
	if finishedAt.IsZero() {
		t.Error("finished_at should be set")
	}

	// Verify pod was unregistered
	if tr.IsPodRegistered("term-pod-1") {
		t.Error("pod should be unregistered from terminal router")
	}

	// Verify callback was called
	if callbackPodKey != "term-pod-1" {
		t.Errorf("callback podKey: got %q, want %q", callbackPodKey, "term-pod-1")
	}
	if callbackStatus != agentpod.StatusCompleted {
		t.Errorf("callback status: got %q, want %q", callbackStatus, agentpod.StatusCompleted)
	}
}
