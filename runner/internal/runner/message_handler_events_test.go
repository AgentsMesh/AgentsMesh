package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for event sending methods and helper functions

// --- Test event sending methods ---

func TestSendPodCreated(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.sendPodCreated("pod-1", 12345, "/worktree/path", "feature/test", 80, 24)

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != client.MsgTypePodCreated {
		t.Errorf("event type = %s, want pod_created", events[0].Type)
	}

	// Mock stores data as map[string]interface{}
	event, ok := events[0].Data.(map[string]interface{})
	if !ok {
		t.Fatalf("event data should be map[string]interface{}")
	}
	if event["pod_key"] != "pod-1" {
		t.Errorf("pod_key = %v, want pod-1", event["pod_key"])
	}
	if event["pid"] != int32(12345) {
		t.Errorf("pid = %v, want 12345", event["pid"])
	}
}

func TestSendPodTerminated(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.sendPodTerminated("pod-1")

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != client.MsgTypePodTerminated {
		t.Errorf("event type = %s, want pod_terminated", events[0].Type)
	}
}

func TestSendTerminalOutput(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.sendTerminalOutput("pod-1", []byte("hello world"))

	// Terminal output uses SendTerminalOutput, check Events
	events := mockConn.GetEvents()
	hasOutput := false
	for _, e := range events {
		if e.Type == client.MsgTypeTerminalOutput {
			hasOutput = true
			break
		}
	}
	if !hasOutput {
		t.Error("should have sent terminal_output message")
	}
}

func TestSendPtyResized(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.sendPtyResized("pod-1", 100, 30)

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != client.MsgTypePtyResized {
		t.Errorf("event type = %s, want pty_resized", events[0].Type)
	}
}

func TestSendPodError(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.sendPodError("pod-1", "something went wrong")

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

// --- Test send methods with nil connection ---

func TestSendMethodsWithNilConnection(t *testing.T) {
	store := NewInMemoryPodStore()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, nil)

	// These should not panic with nil connection
	handler.sendPodCreated("pod-1", 123, "", "", 80, 24)
	handler.sendPodTerminated("pod-1")
	handler.sendTerminalOutput("pod-1", []byte("data"))
	handler.sendPtyResized("pod-1", 80, 24)
	handler.sendPodError("pod-1", "error")
}

// --- Test createOutputHandler ---

func TestCreateOutputHandler(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	outputHandler := handler.createOutputHandler("pod-1")

	// Call the handler
	outputHandler([]byte("test output"))

	// Verify output was sent
	events := mockConn.GetEvents()
	hasOutput := false
	for _, e := range events {
		if e.Type == client.MsgTypeTerminalOutput {
			hasOutput = true
			break
		}
	}
	if !hasOutput {
		t.Error("output handler should send terminal output")
	}
}

// --- Test createExitHandler ---

func TestCreateExitHandler(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pod
	store.Put("exit-pod", &Pod{
		ID:     "exit-pod",
		Status: PodStatusRunning,
	})

	exitHandler := handler.createExitHandler("exit-pod")

	// Call the handler
	exitHandler(0)

	// Verify pod was removed
	_, exists := store.Get("exit-pod")
	if exists {
		t.Error("pod should be removed after exit")
	}

	// Verify terminated event was sent
	events := mockConn.GetEvents()
	hasTerminated := false
	for _, e := range events {
		if e.Type == client.MsgTypePodTerminated {
			hasTerminated = true
			break
		}
	}
	if !hasTerminated {
		t.Error("exit handler should send pod_terminated")
	}
}

// Note: runPreparationScript and mergeEnvVars have been moved to PodBuilder.
// Tests for these functions are in pod_builder_test.go.

// --- Benchmark tests ---

func BenchmarkOnListPods(b *testing.B) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add some pods
	for i := 0; i < 100; i++ {
		store.Put(string(rune('a'+i%26))+string(rune(i)), &Pod{
			ID:     string(rune('a' + i%26)),
			Status: PodStatusRunning,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.OnListPods()
	}
}

// Note: BenchmarkMergeEnvVars moved to pod_builder_test.go since the method is now on PodBuilder.
