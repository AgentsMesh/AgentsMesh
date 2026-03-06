package mcp

import (
	"testing"
)

// --- Test Server Basic Operations (from mcp_test.go) ---

func TestNewServer(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
		Args:    []string{"hello"},
		Env:     map[string]string{"KEY": "VALUE"},
	}

	server := NewServer(cfg)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.name != "test-server" {
		t.Errorf("name: got %v, want test-server", server.name)
	}

	if server.command != testDummyCmd() {
		t.Errorf("command: got %v, want %v", server.command, testDummyCmd())
	}

	if len(server.args) != 1 {
		t.Errorf("args length: got %v, want 1", len(server.args))
	}

	if server.env["KEY"] != "VALUE" {
		t.Errorf("env[KEY]: got %v, want VALUE", server.env["KEY"])
	}

	if server.pending == nil {
		t.Error("pending should be initialized")
	}

	if server.tools == nil {
		t.Error("tools should be initialized")
	}

	if server.resources == nil {
		t.Error("resources should be initialized")
	}
}

func TestServerName(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	server := NewServer(cfg)

	if server.Name() != "test-server" {
		t.Errorf("Name(): got %v, want test-server", server.Name())
	}
}

func TestServerIsRunningInitially(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	server := NewServer(cfg)

	if server.IsRunning() {
		t.Error("server should not be running initially")
	}
}

func TestServerGetToolsEmpty(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	server := NewServer(cfg)

	tools := server.GetTools()
	if len(tools) != 0 {
		t.Errorf("tools should be empty, got %v", len(tools))
	}
}

func TestServerGetResourcesEmpty(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	server := NewServer(cfg)

	resources := server.GetResources()
	if len(resources) != 0 {
		t.Errorf("resources should be empty, got %v", len(resources))
	}
}

func TestServerStopNotRunning(t *testing.T) {
	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	server := NewServer(cfg)

	// Should not error when stopping not running server
	err := server.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
