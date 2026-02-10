package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestHandlePodError(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner with 1 pod
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "error-node",
		Status:         "online",
		CurrentPods:    1,
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create an initializing pod
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"error-pod-1", r.ID, agentpod.StatusInitializing)

	// Track status change callback
	var callbackPodKey, callbackStatus string
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackPodKey = podKey
		callbackStatus = status
	})

	// Handle pod error event
	data := &runnerv1.ErrorEvent{
		PodKey:  "error-pod-1",
		Code:    "GIT_AUTH_FAILED",
		Message: "authentication failed for https://github.com/org/repo.git",
	}

	pc.handlePodError(r.ID, data)

	// Verify pod was updated to error status
	var status string
	var errorCode, errorMessage *string
	pc.db.Raw(`SELECT status, error_code, error_message FROM pods WHERE pod_key = ?`, "error-pod-1").
		Row().Scan(&status, &errorCode, &errorMessage)

	if status != agentpod.StatusError {
		t.Errorf("status: got %q, want %q", status, agentpod.StatusError)
	}
	if errorCode == nil || *errorCode != "GIT_AUTH_FAILED" {
		t.Errorf("error_code: got %v, want %q", errorCode, "GIT_AUTH_FAILED")
	}
	if errorMessage == nil || *errorMessage != "authentication failed for https://github.com/org/repo.git" {
		t.Errorf("error_message: got %v, want %q", errorMessage, "authentication failed for https://github.com/org/repo.git")
	}

	// Verify finished_at was set
	var finishedAt *string
	pc.db.Raw(`SELECT finished_at FROM pods WHERE pod_key = ?`, "error-pod-1").Scan(&finishedAt)
	if finishedAt == nil {
		t.Error("finished_at should be set")
	}

	// Verify callback was called
	if callbackPodKey != "error-pod-1" {
		t.Errorf("callback podKey: got %q, want %q", callbackPodKey, "error-pod-1")
	}
	if callbackStatus != agentpod.StatusError {
		t.Errorf("callback status: got %q, want %q", callbackStatus, agentpod.StatusError)
	}
}

func TestHandlePodError_EmptyPodKey(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Track status change callback - should NOT be called
	callbackCalled := false
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackCalled = true
	})

	// Handle pod error with empty pod_key
	data := &runnerv1.ErrorEvent{
		PodKey:  "",
		Code:    "UNKNOWN",
		Message: "some error",
	}

	// Should not panic, should just log and return
	pc.handlePodError(1, data)

	if callbackCalled {
		t.Error("callback should not be called when pod_key is empty")
	}
}

func TestHandlePodError_NonInitializingPod(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "skip-node",
		Status:         "online",
		CurrentPods:    1,
	}
	if err := pc.db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Create a running pod (NOT initializing)
	pc.db.Exec(`INSERT INTO pods (pod_key, runner_id, status) VALUES (?, ?, ?)`,
		"running-pod-1", r.ID, agentpod.StatusRunning)

	// Track callback
	callbackCalled := false
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackCalled = true
	})

	// Handle pod error event for a running pod
	data := &runnerv1.ErrorEvent{
		PodKey:  "running-pod-1",
		Code:    "PROCESS_CRASH",
		Message: "agent process crashed",
	}

	pc.handlePodError(r.ID, data)

	// Verify pod status was NOT changed (WHERE clause restricts to initializing)
	var status string
	pc.db.Raw(`SELECT status FROM pods WHERE pod_key = ?`, "running-pod-1").Scan(&status)

	if status != agentpod.StatusRunning {
		t.Errorf("status should remain %q, got %q", agentpod.StatusRunning, status)
	}

	// Callback should NOT be called because RowsAffected == 0
	if callbackCalled {
		t.Error("callback should not be called for non-initializing pods")
	}
}

func TestHandlePodError_NonExistentPod(t *testing.T) {
	pc, _, _ := setupPodEventHandlerDeps(t)

	// Track callback
	callbackCalled := false
	pc.SetStatusChangeCallback(func(podKey string, status string, agentStatus string) {
		callbackCalled = true
	})

	// Handle error for a pod that doesn't exist
	data := &runnerv1.ErrorEvent{
		PodKey:  "nonexistent-pod",
		Code:    "GIT_AUTH_FAILED",
		Message: "auth failed",
	}

	// Should not panic
	pc.handlePodError(1, data)

	// Callback should NOT be called because RowsAffected == 0
	if callbackCalled {
		t.Error("callback should not be called for non-existent pods")
	}
}
