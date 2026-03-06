package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// mockMCPServerScript returns a bash script that simulates an MCP server
// Uses Python for better JSON parsing and ID matching
const mockMCPServerScript = `#!/usr/bin/env python3
import sys
import json

# Unbuffered stdin/stdout
sys.stdin = open(sys.stdin.fileno(), 'r', buffering=1)
sys.stdout = open(sys.stdout.fileno(), 'w', buffering=1)

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue

    try:
        req = json.loads(line)
        method = req.get("method", "")
        req_id = req.get("id")

        if method == "initialize":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {
                        "tools": {"listChanged": True},
                        "resources": {"listChanged": True}
                    },
                    "serverInfo": {"name": "mock", "version": "1.0.0"}
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "notifications/initialized":
            # Notification - no response
            pass
        elif method == "tools/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "tools": [{
                        "name": "test_tool",
                        "description": "A test tool",
                        "inputSchema": {"type": "object"}
                    }]
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "resources": [{
                        "uri": "file:///test",
                        "name": "test",
                        "description": "Test resource"
                    }]
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "tools/call":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "content": [{"type": "text", "text": "Tool result"}],
                    "isError": False
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/read":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "contents": [{
                        "uri": "file:///test",
                        "mimeType": "text/plain",
                        "text": "Resource content"
                    }]
                }
            }
            print(json.dumps(resp), flush=True)
    except json.JSONDecodeError:
        pass
`

// createMockMCPServer creates a temporary mock MCP server script
func createMockMCPServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_mcp.py")

	if err := os.WriteFile(scriptPath, []byte(mockMCPServerScript), 0755); err != nil {
		t.Fatalf("failed to create mock MCP server: %v", err)
	}

	return scriptPath
}

// TestServerWithMockMCP tests the full MCP server lifecycle with a mock server
func TestServerWithMockMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)
	pythonCmd := testutil.PythonCommand()

	scriptPath := createMockMCPServer(t)

	server := NewServer(&Config{
		Name:    "mock-test",
		Command: pythonCmd,
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the server
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Verify server is running
	if !server.IsRunning() {
		t.Error("server should be running")
	}

	// Verify tools were loaded
	tools := server.GetTools()
	if len(tools) != 1 {
		t.Errorf("tools count: got %v, want 1", len(tools))
	}

	// Verify resources were loaded
	resources := server.GetResources()
	if len(resources) != 1 {
		t.Errorf("resources count: got %v, want 1", len(resources))
	}
}

// TestServerCallToolWithMockMCP tests calling a tool with a mock server
func TestServerCallToolWithMockMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)
	pythonCmd := testutil.PythonCommand()

	scriptPath := createMockMCPServer(t)

	server := NewServer(&Config{
		Name:    "mock-test",
		Command: pythonCmd,
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the server
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Call a tool
	result, err := server.CallTool(ctx, "test_tool", map[string]interface{}{"arg": "value"})
	if err != nil {
		t.Errorf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Error("result should not be nil")
	}
}

// TestServerReadResourceWithMockMCP tests reading a resource with a mock server
func TestServerReadResourceWithMockMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)
	pythonCmd := testutil.PythonCommand()

	scriptPath := createMockMCPServer(t)

	server := NewServer(&Config{
		Name:    "mock-test",
		Command: pythonCmd,
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the server
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Read a resource
	content, mimeType, err := server.ReadResource(ctx, "file:///test")
	if err != nil {
		t.Errorf("ReadResource failed: %v", err)
	}

	if string(content) != "Resource content" {
		t.Errorf("content: got %v, want 'Resource content'", string(content))
	}

	if mimeType != "text/plain" {
		t.Errorf("mimeType: got %v, want 'text/plain'", mimeType)
	}
}

// TestManagerWithMockMCP tests manager operations with mock servers
func TestManagerWithMockMCP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)
	pythonCmd := testutil.PythonCommand()

	scriptPath := createMockMCPServer(t)

	manager := NewManager()
	manager.AddServer(&Config{
		Name:    "mock-test",
		Command: pythonCmd,
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start all servers
	err := manager.StartAll(ctx)
	if err != nil {
		t.Fatalf("StartAll failed: %v", err)
	}
	defer manager.StopAll()

	// Verify tools
	tools := manager.GetAllTools()
	if len(tools) != 1 {
		t.Errorf("tools map count: got %v, want 1", len(tools))
	}

	// Call tool through manager
	result, err := manager.CallTool(ctx, "mock-test", "test_tool", nil)
	if err != nil {
		t.Errorf("CallTool failed: %v", err)
	}

	if result == nil {
		t.Error("result should not be nil")
	}

	// Read resource through manager
	content, _, err := manager.ReadResource(ctx, "mock-test", "file:///test")
	if err != nil {
		t.Errorf("ReadResource failed: %v", err)
	}

	if string(content) != "Resource content" {
		t.Errorf("content: got %v, want 'Resource content'", string(content))
	}
}
