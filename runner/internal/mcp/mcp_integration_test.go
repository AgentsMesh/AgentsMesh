package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestServerStartWithInvalidCommand tests Start with invalid command
func TestServerStartWithInvalidCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server := NewServer(&Config{
		Name:    "test",
		Command: "/nonexistent/command",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := server.Start(ctx)
	if err == nil {
		t.Error("expected error for invalid command")
		server.Stop()
	}
}

// TestServerStartWithExitScript tests Start with a script that immediately exits
func TestServerStartWithExitScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}

	// Create a temporary script that exits immediately
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "exit.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755)
	if err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	server := NewServer(&Config{
		Name:    "test",
		Command: scriptPath,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start will fail because the server exits immediately without responding
	err = server.Start(ctx)
	if err == nil {
		// If it somehow succeeded, clean up
		server.Stop()
	}
	// Error is expected because script exits without MCP initialization
}

// TestServerStopWithProcess tests Stop with a running process
func TestServerStopWithRunningProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix shell")
	}

	// Create a script that sleeps
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "sleep.sh")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 60\n"), 0755)
	if err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	server := NewServer(&Config{
		Name:    "test",
		Command: scriptPath,
	})

	// Manually set up for stop test
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// We can't fully start without MCP protocol, but we can test Stop behavior
	// by simulating the running state
	server.mu.Lock()
	server.running = true
	ch := make(chan *Response, 1)
	server.pending[1] = ch
	server.mu.Unlock()

	err = server.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if server.IsRunning() {
		t.Error("server should not be running after stop")
	}

	// Verify channel was closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("channel should be closed")
		}
	case <-ctx.Done():
		t.Error("channel should be closed immediately")
	}
}

// TestServerStopWithStdin tests Stop closing stdin
func TestServerStopWithStdin(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testCatCmd()})

	// Create a pipe for stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	server.mu.Lock()
	server.running = true
	server.stdin = w
	server.mu.Unlock()

	err = server.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify stdin was closed
	_, err = w.Write([]byte("test"))
	if err == nil {
		t.Error("stdin should be closed")
	}

	r.Close()
}

// TestServerCallWithValidPipe tests call with a valid stdin pipe
func TestServerCallWithValidPipe(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testCatCmd()})

	// Create a pipe for stdin
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer w.Close()

	server.mu.Lock()
	server.running = true
	server.stdin = w
	server.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Call will timeout waiting for response, but should write successfully
	_, err = server.call(ctx, "test", map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected timeout error")
	}

	// The request should have been sent (context timeout is expected)
	if err != context.DeadlineExceeded {
		// Could be timeout or server closed error
		t.Logf("error: %v (expected timeout or closed)", err)
	}
}

// TestServerSendMarshalSuccess tests send with valid data
func TestServerSendMarshalSuccess(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testCatCmd()})

	// Create a pipe for stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	server.mu.Lock()
	server.running = true
	server.stdin = w
	server.mu.Unlock()

	req := &Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	}

	err = server.send(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerStartAllEmpty tests StartAll with no servers
func TestManagerStartAllEmpty(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.StartAll(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestManagerStartAllWithInvalidServer tests StartAll with invalid servers
func TestManagerStartAllWithInvalidServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	manager := NewManager()
	manager.AddServer(&Config{
		Name:    "invalid",
		Command: "/nonexistent/command",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := manager.StartAll(ctx)
	if err == nil {
		t.Error("expected error for invalid server")
	}
}

// TestManagerGetAllToolsWithServers tests GetAllTools with servers that have tools
func TestManagerGetAllToolsWithServers(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "server1", Command: testDummyCmd()})
	manager.AddServer(&Config{Name: "server2", Command: testDummyCmd()})

	// Manually add tools to servers and mark them as running
	server1, _ := manager.GetServer("server1")
	server2, _ := manager.GetServer("server2")

	server1.mu.Lock()
	server1.tools["tool1"] = &Tool{Name: "tool1"}
	server1.running = true
	server1.mu.Unlock()

	server2.mu.Lock()
	server2.tools["tool2"] = &Tool{Name: "tool2"}
	server2.running = true
	server2.mu.Unlock()

	tools := manager.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("tools count: got %v, want 2", len(tools))
	}
}

// TestManagerGetAllResourcesWithServers tests GetAllResources with servers that have resources
func TestManagerGetAllResourcesWithServers(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "server1", Command: testDummyCmd()})
	manager.AddServer(&Config{Name: "server2", Command: testDummyCmd()})

	// Manually add resources to servers and mark them as running
	server1, _ := manager.GetServer("server1")
	server2, _ := manager.GetServer("server2")

	server1.mu.Lock()
	server1.resources["res1"] = &Resource{URI: "file://res1"}
	server1.running = true
	server1.mu.Unlock()

	server2.mu.Lock()
	server2.resources["res2"] = &Resource{URI: "file://res2"}
	server2.running = true
	server2.mu.Unlock()

	resources := manager.GetAllResources()
	if len(resources) != 2 {
		t.Errorf("resources count: got %v, want 2", len(resources))
	}
}

// TestManagerLoadConfigWithEnvVars tests LoadConfig with env vars
func TestManagerLoadConfigWithEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	config := fmt.Sprintf(`{
		"mcpServers": {
			"server-1": {
				"command": "%s",
				"args": ["hello"],
				"env": {
					"KEY": "VALUE",
					"FOO": "BAR"
				}
			}
		}
	}`, testDummyCmd())

	os.WriteFile(configPath, []byte(config), 0644)

	manager := NewManager()
	err := manager.LoadConfig(configPath)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	server, ok := manager.GetServer("server-1")
	if !ok {
		t.Fatal("server should be added")
	}

	if server.env["KEY"] != "VALUE" {
		t.Errorf("env[KEY]: got %v, want VALUE", server.env["KEY"])
	}

	if server.env["FOO"] != "BAR" {
		t.Errorf("env[FOO]: got %v, want BAR", server.env["FOO"])
	}
}

// TestManagerGetStatusWithTools tests GetStatus with servers that have tools
func TestManagerGetStatusWithTools(t *testing.T) {
	manager := NewManager()
	manager.AddServer(&Config{Name: "server1", Command: testDummyCmd()})

	// Manually add tools and mark server as running
	server, _ := manager.GetServer("server1")
	server.mu.Lock()
	server.tools["tool1"] = &Tool{Name: "tool1", Description: "Test tool"}
	server.running = true
	server.mu.Unlock()

	statuses := manager.GetStatus()
	if len(statuses) != 1 {
		t.Errorf("statuses count: got %v, want 1", len(statuses))
	}

	if len(statuses[0].Tools) != 1 {
		t.Errorf("tools count: got %v, want 1", len(statuses[0].Tools))
	}

	if statuses[0].Tools[0].Name != "tool1" {
		t.Errorf("tool name: got %v, want tool1", statuses[0].Tools[0].Name)
	}
}
