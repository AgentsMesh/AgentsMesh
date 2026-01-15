package runner

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/anthropics/agentmesh/runner/internal/client"
	"github.com/anthropics/agentmesh/runner/internal/config"
)

// Tests for terminal operations: OnTerminalInput, OnTerminalResize

// --- OnTerminalInput Tests ---

func TestOnTerminalInputPodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.TerminalInputRequest{
		PodKey: "nonexistent",
		Data:      base64.StdEncoding.EncodeToString([]byte("hello")),
	}

	err := handler.OnTerminalInput(req)
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}

func TestOnTerminalInputInvalidBase64(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod with nil terminal (will fail on Write, not decode)
	store.Put("input-pod", &Pod{
		ID:       "input-pod",
		Terminal: nil,
	})

	req := client.TerminalInputRequest{
		PodKey: "input-pod",
		Data:      "not-valid-base64!!!", // Invalid base64
	}

	err := handler.OnTerminalInput(req)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestOnTerminalInputSuccess(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// First create a pod
	createReq := client.CreatePodRequest{
		PodKey:      "input-success-pod",
		LaunchCommand: "cat",
	}

	err := handler.OnCreatePod(createReq)
	if err != nil {
		t.Skipf("Could not create pod: %v", err)
	}

	// Wait for terminal to be ready
	time.Sleep(100 * time.Millisecond)

	// Send input
	inputReq := client.TerminalInputRequest{
		PodKey: "input-success-pod",
		Data:      base64.StdEncoding.EncodeToString([]byte("hello\n")),
	}

	err = handler.OnTerminalInput(inputReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Clean up
	pod, ok := store.Get("input-success-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}

// --- OnTerminalResize Tests ---

func TestOnTerminalResizePodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	req := client.TerminalResizeRequest{
		PodKey: "nonexistent",
		Rows:      30,
		Cols:      100,
	}

	err := handler.OnTerminalResize(req)
	if err == nil {
		t.Error("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}

func TestOnTerminalResizeSuccess(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:         tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// First create a pod
	createReq := client.CreatePodRequest{
		PodKey:      "resize-pod",
		LaunchCommand: "cat",
	}

	err := handler.OnCreatePod(createReq)
	if err != nil {
		t.Skipf("Could not create pod: %v", err)
	}

	// Wait for terminal to be ready
	time.Sleep(100 * time.Millisecond)

	// Now resize the terminal
	resizeReq := client.TerminalResizeRequest{
		PodKey: "resize-pod",
		Rows:      40,
		Cols:      120,
	}

	err = handler.OnTerminalResize(resizeReq)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify pty_resized event was sent
	events := mockConn.GetEvents()
	hasResized := false
	for _, e := range events {
		if e.Type == client.MsgTypePtyResized {
			hasResized = true
			break
		}
	}
	if !hasResized {
		t.Error("should have sent pty_resized event")
	}

	// Clean up
	pod, ok := store.Get("resize-pod")
	if ok && pod.Terminal != nil {
		pod.Terminal.Stop()
	}
}
