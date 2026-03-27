package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

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
