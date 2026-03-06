package mcp

import (
	"encoding/json"
	"testing"
)

// --- Test ServerStatus Struct ---

func TestServerStatusStruct(t *testing.T) {
	status := ServerStatus{
		Name:    "test-server",
		Running: true,
		Tools: []*Tool{
			{Name: "tool1", Description: "Tool 1"},
		},
		Resources: []*Resource{
			{URI: "file://test.txt", Name: "test.txt"},
		},
	}

	if status.Name != "test-server" {
		t.Errorf("Name: got %v, want test-server", status.Name)
	}

	if !status.Running {
		t.Error("Running should be true")
	}

	if len(status.Tools) != 1 {
		t.Errorf("Tools count: got %v, want 1", len(status.Tools))
	}

	if len(status.Resources) != 1 {
		t.Errorf("Resources count: got %v, want 1", len(status.Resources))
	}
}

func TestServerStatusJSON(t *testing.T) {
	status := ServerStatus{
		Name:    "test-server",
		Running: false,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var unmarshaled ServerStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != "test-server" {
		t.Errorf("Name: got %v, want test-server", unmarshaled.Name)
	}
}

// --- Benchmark Tests ---

func BenchmarkNewManager(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewManager()
	}
}

func BenchmarkManagerAddServer(b *testing.B) {
	manager := NewManager()
	cfg := &Config{Name: "test", Command: testDummyCmd()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.AddServer(cfg)
	}
}

func BenchmarkManagerListServers(b *testing.B) {
	manager := NewManager()
	for i := 0; i < 10; i++ {
		manager.AddServer(&Config{
			Name:    "server-" + string(rune('0'+i)),
			Command: testDummyCmd(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ListServers()
	}
}

func BenchmarkManagerGetServer(b *testing.B) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "test-server", Command: testDummyCmd()})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetServer("test-server")
	}
}

func BenchmarkNewServer(b *testing.B) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
		Args:    []string{"hello"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewServer(cfg)
	}
}
