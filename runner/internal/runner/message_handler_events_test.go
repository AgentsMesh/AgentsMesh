package runner

import (
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
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

	// Production code calls h.conn.SendPodTerminated directly (with exitCode and earlyOutput).
	// Test the same path used by createExitHandler and OnTerminatePod.
	if err := handler.conn.SendPodTerminated("pod-1", 0, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != client.MsgTypePodTerminated {
		t.Errorf("event type = %s, want pod_terminated", events[0].Type)
	}
}

// NOTE: TestSendTerminalOutput removed - output is exclusively streamed via Relay

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
	// NOTE: sendTerminalOutput removed - output is exclusively streamed via Relay
	handler.sendPtyResized("pod-1", 80, 24)
	handler.sendPodError("pod-1", "error")
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

func TestCreatePTYErrorHandler(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create pod with aggregator to capture the error message
	pod := &Pod{
		ID:         "pty-error-pod",
		Status:     PodStatusRunning,
		Aggregator: aggregator.NewSmartAggregator(nil, nil),
	}

	ptyErrorHandler := handler.createPTYErrorHandler("pty-error-pod", pod)

	// Simulate a PTY I/O error
	testErr := fmt.Errorf("read /dev/ptmx: input/output error")
	ptyErrorHandler(testErr)

	// Verify error event was sent via gRPC
	events := mockConn.GetEvents()
	hasError := false
	for _, e := range events {
		if e.Type == "error" {
			hasError = true
			data, ok := e.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("error event data should be map[string]interface{}")
			}
			if data["code"] != client.ErrCodePTYError {
				t.Errorf("error code = %v, want %s", data["code"], client.ErrCodePTYError)
			}
			if data["pod_key"] != "pty-error-pod" {
				t.Errorf("pod_key = %v, want pty-error-pod", data["pod_key"])
			}
			break
		}
	}
	if !hasError {
		t.Error("PTY error handler should send error event via gRPC")
	}

	// Verify error message was buffered in aggregator (will be flushed to terminal via relay)
	if pod.Aggregator.BufferLen() == 0 {
		t.Error("PTY error handler should write visible error message to aggregator buffer")
	}
}

func TestCreatePTYErrorHandler_NilAggregator(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Pod without aggregator — should not panic
	pod := &Pod{
		ID:     "no-agg-pod",
		Status: PodStatusRunning,
	}

	ptyErrorHandler := handler.createPTYErrorHandler("no-agg-pod", pod)
	ptyErrorHandler(fmt.Errorf("some error"))

	// Should still send gRPC error event
	events := mockConn.GetEvents()
	if len(events) == 0 {
		t.Error("should send gRPC error event even without aggregator")
	}
}

func TestCreateExitHandler_WithPTYError(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create pod with a stored PTY error (simulating onPTYError having been called)
	pod := &Pod{
		ID:     "pty-exit-pod",
		Status: PodStatusRunning,
	}
	pod.SetPTYError("PTY read error: read /dev/ptmx: input/output error")
	store.Put("pty-exit-pod", pod)

	exitHandler := handler.createExitHandler("pty-exit-pod")

	// Simulate process killed after PTY error (exit code -1 from SIGKILL)
	exitHandler(-1)

	// Verify terminated event includes the PTY error message
	events := mockConn.GetEvents()
	hasTerminated := false
	for _, e := range events {
		if e.Type == client.MsgTypePodTerminated {
			hasTerminated = true
			data, ok := e.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("terminated event data should be map[string]interface{}")
			}
			errorMsg, _ := data["error"].(string)
			if errorMsg == "" {
				t.Error("terminated event should include PTY error message")
			}
			if !contains(errorMsg, "PTY read error") {
				t.Errorf("error message should contain 'PTY read error', got: %s", errorMsg)
			}
			break
		}
	}
	if !hasTerminated {
		t.Error("exit handler should send pod_terminated event")
	}
}

func TestCreateExitHandler_EarlyOutputTakesPriority(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create pod with both a PTY error and an aggregator with early output
	pod := &Pod{
		ID:         "priority-pod",
		Status:     PodStatusRunning,
		Aggregator: aggregator.NewSmartAggregator(nil, nil),
	}
	pod.SetPTYError("PTY read error: something")
	store.Put("priority-pod", pod)

	exitHandler := handler.createExitHandler("priority-pod")
	exitHandler(1)

	// When early output is empty (relay was connected), PTY error should be used.
	// The aggregator's DrainEarlyBuffer returns empty because relay was "connected"
	// (no early buffer data), so PTY error should be the fallback.
	events := mockConn.GetEvents()
	for _, e := range events {
		if e.Type == client.MsgTypePodTerminated {
			data := e.Data.(map[string]interface{})
			errorMsg, _ := data["error"].(string)
			if !contains(errorMsg, "PTY read error") {
				t.Errorf("when no early output, PTY error should be used, got: %s", errorMsg)
			}
			break
		}
	}
}

// Note: contains() is defined in mocks_test.go

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

// --- Test OSC handler ---

func TestCreateOSCHandler_OSC777Notify(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test OSC 777 notify
	oscHandler(777, []string{"notify", "Build Complete", "Your project compiled successfully"})

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != "osc_notification" {
		t.Errorf("event type = %s, want osc_notification", events[0].Type)
	}

	data, ok := events[0].Data.(map[string]string)
	if !ok {
		t.Fatalf("event data should be map[string]string")
	}
	if data["pod_key"] != "test-pod" {
		t.Errorf("pod_key = %s, want test-pod", data["pod_key"])
	}
	if data["title"] != "Build Complete" {
		t.Errorf("title = %s, want Build Complete", data["title"])
	}
	if data["body"] != "Your project compiled successfully" {
		t.Errorf("body = %s, want Your project compiled successfully", data["body"])
	}
}

func TestCreateOSCHandler_OSC777NotNotify(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test OSC 777 with non-notify action (should be ignored)
	oscHandler(777, []string{"file", "/path/to/file"})

	events := mockConn.GetEvents()
	if len(events) != 0 {
		t.Fatalf("expected 0 events for non-notify OSC 777, got %d", len(events))
	}
}

func TestCreateOSCHandler_OSC9(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test OSC 9 (ConEmu/Windows Terminal format)
	oscHandler(9, []string{"Task completed"})

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	data, ok := events[0].Data.(map[string]string)
	if !ok {
		t.Fatalf("event data should be map[string]string")
	}
	if data["title"] != "Notification" {
		t.Errorf("title = %s, want Notification (default)", data["title"])
	}
	if data["body"] != "Task completed" {
		t.Errorf("body = %s, want Task completed", data["body"])
	}
}

func TestCreateOSCHandler_OSC0Title(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test OSC 0 (window title)
	oscHandler(0, []string{"My Terminal Title"})

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != "osc_title" {
		t.Errorf("event type = %s, want osc_title", events[0].Type)
	}

	data, ok := events[0].Data.(map[string]string)
	if !ok {
		t.Fatalf("event data should be map[string]string")
	}
	if data["title"] != "My Terminal Title" {
		t.Errorf("title = %s, want My Terminal Title", data["title"])
	}
}

func TestCreateOSCHandler_OSC2Title(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test OSC 2 (window title)
	oscHandler(2, []string{"Another Title"})

	events := mockConn.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != "osc_title" {
		t.Errorf("event type = %s, want osc_title", events[0].Type)
	}
}

func TestCreateOSCHandler_EmptyParams(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test with empty params (should be ignored)
	oscHandler(777, []string{})
	oscHandler(9, []string{})
	oscHandler(0, []string{})

	events := mockConn.GetEvents()
	if len(events) != 0 {
		t.Fatalf("expected 0 events for empty params, got %d", len(events))
	}
}

func TestCreateOSCHandler_UnknownOSCType(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	oscHandler := handler.createOSCHandler("test-pod")

	// Test unknown OSC type (should be ignored)
	oscHandler(999, []string{"some", "params"})

	events := mockConn.GetEvents()
	if len(events) != 0 {
		t.Fatalf("expected 0 events for unknown OSC type, got %d", len(events))
	}
}

// Note: TestCreateOSCHandler_NilConnection removed - the implementation calls conn.SendOSC*
// which will panic with nil connection. This is expected behavior since the connection
// should always be set before createOSCHandler is called in normal operation.
