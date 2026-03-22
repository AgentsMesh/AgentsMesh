package acp

import (
	"log/slog"
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
