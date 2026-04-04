package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for OnListPods operations

func TestOnListPodsEmpty(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pods := handler.OnListPods()
	if len(pods) != 0 {
		t.Errorf("expected 0 pods, got %d", len(pods))
	}
}

func TestOnListPodsWithPods(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pods
	store.Put("pod-1", &Pod{
		ID:     "pod-1",
		PodKey: "pod-1",
		Status: PodStatusRunning,
	})
	store.Put("pod-2", &Pod{
		ID:     "pod-2",
		PodKey: "pod-2",
		Status: PodStatusInitializing,
	})

	pods := handler.OnListPods()
	if len(pods) != 2 {
		t.Errorf("expected 2 pods, got %d", len(pods))
	}
}

func TestOnListPodsWithTerminalPID(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// First create a pod with a real terminal
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "list-pid-pod",
		LaunchCommand: "sleep",
		LaunchArgs:    []string{"60"},
		AgentfileSource: "AGENT sleep\nPROMPT_POSITION prepend\n",
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Skipf("Could not create pod: %v", err)
	}

	// List pods
	pods := handler.OnListPods()
	if len(pods) != 1 {
		t.Fatalf("pods count = %d, want 1", len(pods))
	}

	// Check PID is set
	if pods[0].Pid == 0 {
		t.Error("Pod PID should be non-zero")
	}

	// Clean up
	pod, ok := store.Get("list-pid-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}

// Note: Helper function contains() is in mocks_test.go
