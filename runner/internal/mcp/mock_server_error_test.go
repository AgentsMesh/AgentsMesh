package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// mockMCPErrorServerScript returns a script that returns errors
const mockMCPErrorServerScript = `#!/usr/bin/env python3
import sys
import json

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
                    "capabilities": {"tools": {"listChanged": True}},
                    "serverInfo": {"name": "mock", "version": "1.0.0"}
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "notifications/initialized":
            pass
        elif method == "tools/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {"tools": []}
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {"resources": []}
            }
            print(json.dumps(resp), flush=True)
        elif method == "tools/call":
            # Return error response
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "error": {
                    "code": -32600,
                    "message": "Tool execution failed"
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/read":
            # Return error response
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "error": {
                    "code": -32600,
                    "message": "Resource read failed"
                }
            }
            print(json.dumps(resp), flush=True)
    except json.JSONDecodeError:
        pass
`

// mockMCPIsErrorServerScript returns a script that returns isError in tool call
const mockMCPIsErrorServerScript = `#!/usr/bin/env python3
import sys
import json

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
                    "capabilities": {"tools": {"listChanged": True}},
                    "serverInfo": {"name": "mock", "version": "1.0.0"}
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "notifications/initialized":
            pass
        elif method == "tools/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {"tools": [{"name": "failing_tool", "description": "A tool that fails"}]}
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/list":
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {"resources": []}
            }
            print(json.dumps(resp), flush=True)
        elif method == "tools/call":
            # Return isError in result
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {
                    "content": [{"type": "text", "text": "Tool error message"}],
                    "isError": True
                }
            }
            print(json.dumps(resp), flush=True)
        elif method == "resources/read":
            # Return empty contents
            resp = {
                "jsonrpc": "2.0",
                "id": req_id,
                "result": {"contents": []}
            }
            print(json.dumps(resp), flush=True)
    except json.JSONDecodeError:
        pass
`

func createErrorMCPServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_error_mcp.py")
	if err := os.WriteFile(scriptPath, []byte(mockMCPErrorServerScript), 0755); err != nil {
		t.Fatalf("failed to create mock error MCP server: %v", err)
	}
	return scriptPath
}

func createIsErrorMCPServer(t *testing.T) string {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_iserror_mcp.py")
	if err := os.WriteFile(scriptPath, []byte(mockMCPIsErrorServerScript), 0755); err != nil {
		t.Fatalf("failed to create mock isError MCP server: %v", err)
	}
	return scriptPath
}

// TestCallToolMCPErrorResponse tests CallTool with MCP error response
func TestCallToolMCPErrorResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)

	scriptPath := createErrorMCPServer(t)

	server := NewServer(&Config{
		Name:    "error-test",
		Command: testutil.PythonCommand(),
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Call tool that returns error
	_, err = server.CallTool(ctx, "test_tool", nil)
	if err == nil {
		t.Error("expected error from CallTool")
	}
}

// TestCallToolIsErrorResponse tests CallTool with isError in result
func TestCallToolIsErrorResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)

	scriptPath := createIsErrorMCPServer(t)

	server := NewServer(&Config{
		Name:    "iserror-test",
		Command: testutil.PythonCommand(),
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Call tool that returns isError
	_, err = server.CallTool(ctx, "failing_tool", nil)
	if err == nil {
		t.Error("expected error from CallTool with isError")
	}

	// Error should contain the text from content
	if err != nil && err.Error() != "tool error: Tool error message" {
		t.Logf("error message: %v", err)
	}
}

// TestReadResourceErrorResponse tests ReadResource with error response
func TestReadResourceErrorResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)

	scriptPath := createErrorMCPServer(t)

	server := NewServer(&Config{
		Name:    "error-test",
		Command: testutil.PythonCommand(),
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Read resource that returns error
	_, _, err = server.ReadResource(ctx, "file:///test")
	if err == nil {
		t.Error("expected error from ReadResource")
	}
}

// TestReadResourceEmptyContentsError tests ReadResource with empty contents
func TestReadResourceEmptyContentsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)

	scriptPath := createIsErrorMCPServer(t)

	server := NewServer(&Config{
		Name:    "empty-test",
		Command: testutil.PythonCommand(),
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Read resource that returns empty contents
	_, _, err = server.ReadResource(ctx, "file:///test")
	if err == nil {
		t.Error("expected error for empty contents")
	}
}

// TestCallToolEmptyIsError tests CallTool with isError but empty content
func TestCallToolEmptyIsError(t *testing.T) {
	// Create a script that returns isError with no text
	const emptyIsErrorScript = `#!/usr/bin/env python3
import sys
import json

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
            resp = {"jsonrpc": "2.0", "id": req_id, "result": {"protocolVersion": "2024-11-05", "capabilities": {}, "serverInfo": {"name": "mock", "version": "1.0.0"}}}
            print(json.dumps(resp), flush=True)
        elif method == "notifications/initialized":
            pass
        elif method == "tools/list":
            resp = {"jsonrpc": "2.0", "id": req_id, "result": {"tools": []}}
            print(json.dumps(resp), flush=True)
        elif method == "resources/list":
            resp = {"jsonrpc": "2.0", "id": req_id, "result": {"resources": []}}
            print(json.dumps(resp), flush=True)
        elif method == "tools/call":
            # Return isError with empty content
            resp = {"jsonrpc": "2.0", "id": req_id, "result": {"content": [], "isError": True}}
            print(json.dumps(resp), flush=True)
    except json.JSONDecodeError:
        pass
`

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testutil.SkipIfNoPython(t)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "empty_iserror.py")
	if err := os.WriteFile(scriptPath, []byte(emptyIsErrorScript), 0755); err != nil {
		t.Fatalf("failed to create script: %v", err)
	}

	server := NewServer(&Config{
		Name:    "empty-iserror-test",
		Command: testutil.PythonCommand(),
		Args:    []string{scriptPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	// Call tool that returns isError with empty content
	_, err = server.CallTool(ctx, "test", nil)
	if err == nil {
		t.Error("expected error from CallTool")
	}

	// Should return generic error
	if err != nil && err.Error() != "tool returned error" {
		t.Logf("error message: %v", err)
	}
}
