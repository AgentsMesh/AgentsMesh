package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for terminal operations: OnPodInput

// --- OnPodInput Tests ---

func TestOnPodInputPodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.PodInputRequest{
		PodKey: "nonexistent",
		Data:   []byte("hello"),
	}

	err := handler.OnPodInput(req)
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}

func TestOnPodInputNilTerminal(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod with nil terminal (will fail on Write)
	store.Put("input-pod", &Pod{
		ID: "input-pod",
	})

	req := client.PodInputRequest{
		PodKey: "input-pod",
		Data:   []byte("some input"),
	}

	err := handler.OnPodInput(req)
	if err == nil {
		t.Error("expected error for nil terminal")
	}
}

func TestOnPodInputSuccess(t *testing.T) {
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

	// First create a pod
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "input-success-pod",
		LaunchCommand: "cat",
		AgentfileSource: "AGENT cat\nPROMPT_POSITION prepend\n",
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Skipf("Could not create pod: %v", err)
	}

	// Wait for terminal to be ready
	time.Sleep(100 * time.Millisecond)

	// Send input
	inputReq := client.PodInputRequest{
		PodKey: "input-success-pod",
		Data:   []byte("hello\n"),
	}

	err = handler.OnPodInput(inputReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Clean up
	pod, ok := store.Get("input-success-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}
