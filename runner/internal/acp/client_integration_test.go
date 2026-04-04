package acp

import (
	"log/slog"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestACPClient_StartStop(t *testing.T) {
	// Verify the mock agent binary works
	cmd := exec.Command(mockAgentCmd(), mockAgentArgs()...)
	cmd.Env = mockAgentEnv()
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start mock agent: %v", err)
	}
	cmd.Process.Kill()
	cmd.Wait()

	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		Logger:  slog.Default(),
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// After Start, state should be idle (initialize succeeded)
	if client.State() != StateIdle {
		t.Errorf("expected state idle, got %s", client.State())
	}

	client.Stop()

	// After Stop, state should be stopped
	if client.State() != StateStopped {
		t.Errorf("expected state stopped, got %s", client.State())
	}
}

func TestACPClient_NewSession(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if client.SessionID() != "mock-session-001" {
		t.Errorf("expected session ID 'mock-session-001', got %q", client.SessionID())
	}
}

func TestACPClient_SendPrompt(t *testing.T) {
	var mu sync.Mutex
	var receivedChunks []ContentChunk
	var stateChanges []string

	client := NewClient(ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		Logger:  slog.Default(),
		Callbacks: EventCallbacks{
			OnContentChunk: func(sessionID string, chunk ContentChunk) {
				mu.Lock()
				receivedChunks = append(receivedChunks, chunk)
				mu.Unlock()
			},
			OnStateChange: func(newState string) {
				mu.Lock()
				stateChanges = append(stateChanges, newState)
				mu.Unlock()
			},
		},
	})

	if err := client.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if err := client.SendPrompt("hello"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	// Wait for notifications to arrive
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		chunks := len(receivedChunks)
		mu.Unlock()
		if client.State() == StateIdle && chunks > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for prompt response, state=%s chunks=%d",
				client.State(), chunks)
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Verify content was received
	mu.Lock()
	defer mu.Unlock()
	if len(receivedChunks) == 0 {
		t.Error("expected at least one content chunk")
	} else if receivedChunks[0].Text != "Hello from mock agent" {
		t.Errorf("unexpected chunk text: %q", receivedChunks[0].Text)
	}
}

func TestACPClient_SendPrompt_WrongState(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	client.setState(StateProcessing)

	err := client.SendPrompt("should fail")
	if err == nil {
		t.Error("expected error when sending prompt in processing state")
	}
}

func TestACPClient_RespondToPermission(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	client.setState(StateWaitingPermission)

	err := client.RespondToPermission("123", true)
	if err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	if client.State() != StateProcessing {
		t.Errorf("expected state processing after permission response, got %s", client.State())
	}
}

func TestACPClient_CancelSession(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	err := client.CancelSession()
	if err != nil {
		t.Fatalf("CancelSession: %v", err)
	}
}

func TestACPClient_Done(t *testing.T) {
	client := startMockClient(t)

	done := client.Done()
	select {
	case <-done:
		t.Error("done channel should not be closed while running")
	default:
	}

	client.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("done channel should be closed after Stop")
	}
}

func TestACPClient_GetSessionSnapshot_Integration(t *testing.T) {
	client := startMockClient(t)
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	client.AddPendingPermission(PermissionRequest{
		RequestID: "perm-1",
		ToolName:  "file_write",
	})

	snapshot := client.GetSessionSnapshot()
	if snapshot.SessionID != "mock-session-001" {
		t.Errorf("snapshot session ID: %q", snapshot.SessionID)
	}
	if snapshot.State != StateIdle {
		t.Errorf("snapshot state: %q", snapshot.State)
	}
	if len(snapshot.PendingPermissions) != 1 {
		t.Errorf("expected 1 pending permission, got %d", len(snapshot.PendingPermissions))
	}
}
