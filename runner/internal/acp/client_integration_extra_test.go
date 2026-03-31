package acp

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"testing"
	"time"
)

func TestACPClient_ReadStderr(t *testing.T) {
	var logMessages []string

	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		Logger:  slog.Default(),
		Callbacks: EventCallbacks{
			OnLog: func(level, message string) {
				logMessages = append(logMessages, message)
			},
		},
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Stop()

	// Mock agent doesn't write to stderr, but readStderr is running.
	time.Sleep(100 * time.Millisecond)
}

func TestACPClient_HandleAgentRequest(t *testing.T) {
	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnvWithMode(mockModeSendReq),
		Logger:  slog.Default(),
	})
	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Stop()

	time.Sleep(200 * time.Millisecond)

	if s := client.State(); s != StateIdle {
		t.Errorf("expected idle state after handling agent request, got %s", s)
	}
}

func TestACPClient_DoubleStop(t *testing.T) {
	client := startMockClient(t)

	client.Stop()
	client.Stop()

	if client.State() != StateStopped {
		t.Errorf("expected stopped, got %s", client.State())
	}
}

func TestACPClient_NewSessionWithMCPServers(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	mcpServers := BuildMCPServersConfig(9999)
	if err := client.NewSession(mcpServers); err != nil {
		t.Fatalf("NewSession with MCP: %v", err)
	}
}

// TestMockAgent_SmokeTest ensures the mock agent JSON-RPC exchange is well-formed.
func TestMockAgent_SmokeTest(t *testing.T) {
	cmd := exec.Command(mockAgentCmd(), mockAgentArgs()...)
	cmd.Env = mockAgentEnv()

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer cmd.Process.Kill()

	writer := NewWriter(stdin)
	reader := NewReader(stdout, slog.Default())

	id, _ := writer.WriteRequest("initialize", map[string]any{
		"protocolVersion":    1,
		"clientInfo":         map[string]any{"name": "test", "version": "1.0"},
		"clientCapabilities": map[string]any{},
	})

	msg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("read init response: %v", err)
	}
	if !msg.IsResponse() {
		t.Fatalf("expected response, got method=%s", msg.Method)
	}
	respID, _ := msg.GetID()
	if respID != id {
		t.Errorf("response ID mismatch: %d != %d", respID, id)
	}

	var result map[string]any
	json.Unmarshal(msg.Result, &result)
	if result["protocol_version"] != "2025-01-01" {
		t.Errorf("unexpected protocol version: %v", result["protocol_version"])
	}
}
