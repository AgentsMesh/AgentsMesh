package acp

import (
	"sync"
	"testing"
	"time"
)

// TestACPClient_ValidateMode verifies the validate mock agent accepts
// well-formed initialize, session/new, and session/prompt params.
func TestACPClient_ValidateMode(t *testing.T) {
	var mu sync.Mutex
	var chunks []ContentChunk

	client := startMockClientWithMode(t, mockModeValidate, EventCallbacks{
		OnContentChunk: func(_ string, chunk ContentChunk) {
			mu.Lock()
			chunks = append(chunks, chunk)
			mu.Unlock()
		},
	})
	defer client.Stop()

	// Start succeeded → initialize params were validated (protocolVersion, clientInfo, clientCapabilities).
	if client.State() != StateIdle {
		t.Fatalf("expected idle after validated init, got %s", client.State())
	}

	// NewSession with MCP servers → cwd + mcpServers validated.
	mcpServers := BuildMCPServersConfig(9999)
	if err := client.NewSession(mcpServers); err != nil {
		t.Fatalf("NewSession (validate): %v", err)
	}
	if client.SessionID() != "mock-session-001" {
		t.Errorf("session ID = %q, want mock-session-001", client.SessionID())
	}

	// SendPrompt → sessionId + prompt array validated.
	if err := client.SendPrompt("hello validate"); err != nil {
		t.Fatalf("SendPrompt (validate): %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		got := len(chunks)
		mu.Unlock()
		if got > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for content in validate mode")
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if chunks[0].Text != "Hello from mock agent" {
		t.Errorf("chunk text = %q, want 'Hello from mock agent'", chunks[0].Text)
	}
}
