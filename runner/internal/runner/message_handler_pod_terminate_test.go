package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for OnTerminatePod operations

func TestOnTerminatePodSuccess(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{WorkspaceRoot: tempDir},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod
	store.Put("terminate-pod", &Pod{
		ID: "terminate-pod",
	})

	req := client.TerminatePodRequest{
		PodKey: "terminate-pod",
	}

	err := handler.OnTerminatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify pod was removed
	_, exists := store.Get("terminate-pod")
	if exists {
		t.Error("pod should be removed")
	}

	// Verify pod_terminated event was sent
	events := mockConn.GetEvents()
	hasTerminated := false
	for _, e := range events {
		if e.Type == client.MsgTypePodTerminated {
			hasTerminated = true
			break
		}
	}
	if !hasTerminated {
		t.Error("should have sent pod_terminated event")
	}
}

func TestOnTerminatePodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.TerminatePodRequest{
		PodKey: "nonexistent-pod",
	}

	err := handler.OnTerminatePod(req)
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}

func TestOnTerminatePodWithWorktree(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{WorkspaceRoot: tempDir},
		// No worktreeService
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod with worktree
	store.Put("worktree-pod", &Pod{
		ID:          "worktree-pod",
		SandboxPath: "/fake/worktree/path",
	})

	req := client.TerminatePodRequest{
		PodKey: "worktree-pod",
	}

	err := handler.OnTerminatePod(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
