package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- Test Manager ---

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager returned nil")
		return // unreachable, satisfies staticcheck SA5011
	}

	if manager.servers == nil {
		t.Error("servers should be initialized")
	}
}

func TestManagerAddServer(t *testing.T) {
	manager := NewManager()

	cfg := &Config{
		Name:    "test-server",
		Command: testDummyCmd(),
	}

	manager.AddServer(cfg)

	server, ok := manager.GetServer("test-server")
	if !ok {
		t.Error("server should be added")
	}

	if server.Name() != "test-server" {
		t.Errorf("Name: got %v, want test-server", server.Name())
	}
}

func TestManagerGetServerNotFound(t *testing.T) {
	manager := NewManager()

	_, ok := manager.GetServer("nonexistent")
	if ok {
		t.Error("should return false for nonexistent server")
	}
}

func TestManagerListServers(t *testing.T) {
	manager := NewManager()

	manager.AddServer(&Config{Name: "server-1", Command: testDummyCmd()})
	manager.AddServer(&Config{Name: "server-2", Command: testDummyCmd()})

	servers := manager.ListServers()
	if len(servers) != 2 {
		t.Errorf("servers count: got %v, want 2", len(servers))
	}
}

func TestManagerListServersEmpty(t *testing.T) {
	manager := NewManager()

	servers := manager.ListServers()
	if len(servers) != 0 {
		t.Errorf("servers should be empty, got %v", len(servers))
	}
}

func TestManagerStartServerNotFound(t *testing.T) {
	manager := NewManager()

	err := manager.StartServer(context.TODO(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestManagerStopServerNotFound(t *testing.T) {
	manager := NewManager()

	err := manager.StopServer("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestManagerStopAll(t *testing.T) {
	manager := NewManager()

	manager.AddServer(&Config{Name: "server-1", Command: testDummyCmd()})
	manager.AddServer(&Config{Name: "server-2", Command: testDummyCmd()})

	// Should not panic
	manager.StopAll()
}

func TestManagerGetAllToolsEmpty(t *testing.T) {
	manager := NewManager()

	tools := manager.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("tools should be empty, got %v", len(tools))
	}
}

func TestManagerGetAllResourcesEmpty(t *testing.T) {
	manager := NewManager()

	resources := manager.GetAllResources()
	if len(resources) != 0 {
		t.Errorf("resources should be empty, got %v", len(resources))
	}
}

func TestManagerCallToolServerNotFound(t *testing.T) {
	manager := NewManager()

	_, err := manager.CallTool(context.TODO(), "nonexistent", "tool", nil)
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestManagerCallToolServerNotRunning(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "test-server", Command: testDummyCmd()})

	_, err := manager.CallTool(context.TODO(), "test-server", "tool", nil)
	if err == nil {
		t.Error("expected error for server not running")
	}
}

func TestManagerReadResourceServerNotFound(t *testing.T) {
	manager := NewManager()

	_, _, err := manager.ReadResource(context.TODO(), "nonexistent", "file://test")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestManagerReadResourceServerNotRunning(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "test-server", Command: testDummyCmd()})

	_, _, err := manager.ReadResource(context.TODO(), "test-server", "file://test")
	if err == nil {
		t.Error("expected error for server not running")
	}
}

func TestManagerGetStatus(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "server-1", Command: testDummyCmd()})
	manager.AddServer(&Config{Name: "server-2", Command: testDummyCmd()})

	statuses := manager.GetStatus()
	if len(statuses) != 2 {
		t.Errorf("statuses count: got %v, want 2", len(statuses))
	}

	// All should be not running
	for _, status := range statuses {
		if status.Running {
			t.Errorf("server %s should not be running", status.Name)
		}
	}
}

func TestManagerGetStatusEmpty(t *testing.T) {
	manager := NewManager()

	statuses := manager.GetStatus()
	if len(statuses) != 0 {
		t.Errorf("statuses should be empty, got %v", len(statuses))
	}
}

func TestManagerLoadConfigNonExistent(t *testing.T) {
	manager := NewManager()

	err := manager.LoadConfig("/nonexistent/path/config.json")
	if err != nil {
		t.Errorf("should not error for nonexistent file: %v", err)
	}
}

func TestManagerLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	os.WriteFile(configPath, []byte("invalid json"), 0644)

	manager := NewManager()
	err := manager.LoadConfig(configPath)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestManagerLoadConfigValid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := fmt.Sprintf(`{
		"mcpServers": {
			"server-1": {
				"command": "%s",
				"args": ["hello"]
			},
			"server-2": {
				"command": "%s"
			}
		}
	}`, testDummyCmd(), testCatCmd())

	os.WriteFile(configPath, []byte(config), 0644)

	manager := NewManager()
	err := manager.LoadConfig(configPath)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	servers := manager.ListServers()
	if len(servers) != 2 {
		t.Errorf("servers count: got %v, want 2", len(servers))
	}

	// Verify servers were added
	_, ok1 := manager.GetServer("server-1")
	_, ok2 := manager.GetServer("server-2")

	if !ok1 || !ok2 {
		t.Error("servers should be added from config")
	}
}
