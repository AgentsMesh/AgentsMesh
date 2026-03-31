package acp

import (
	"sync"
	"testing"
	"time"
)

// TestACPClient_PermissionRequest_Deny verifies the permission denial flow:
// Agent sends permission request → Client denies → Agent sends content anyway →
// State transitions: idle → processing → waiting_permission → processing → idle.
func TestACPClient_PermissionRequest_Deny(t *testing.T) {
	var mu sync.Mutex
	var permReqs []PermissionRequest
	var chunks []ContentChunk
	var stateChanges []string

	client := startMockClientWithMode(t, mockModeSendPerm, EventCallbacks{
		OnPermissionRequest: func(req PermissionRequest) {
			mu.Lock()
			permReqs = append(permReqs, req)
			mu.Unlock()
		},
		OnContentChunk: func(_ string, chunk ContentChunk) {
			mu.Lock()
			chunks = append(chunks, chunk)
			mu.Unlock()
		},
		OnStateChange: func(newState string) {
			mu.Lock()
			stateChanges = append(stateChanges, newState)
			mu.Unlock()
		},
	})
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := client.SendPrompt("do something dangerous"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	// Wait for permission request.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		got := len(permReqs) > 0
		mu.Unlock()
		if got {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for permission request")
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	req := permReqs[0]
	mu.Unlock()

	// State should be waiting_permission before denial.
	if client.State() != StateWaitingPermission {
		t.Errorf("pre-deny state = %s, want %s", client.State(), StateWaitingPermission)
	}

	// Deny the permission.
	if err := client.RespondToPermission(req.RequestID, false); err != nil {
		t.Fatalf("RespondToPermission(deny): %v", err)
	}

	// After denial, state should transition to processing (mock agent continues).
	if client.State() != StateProcessing {
		t.Errorf("post-deny state = %s, want %s", client.State(), StateProcessing)
	}

	// Wait for content (mock agent sends content regardless of deny).
	deadline2 := time.After(5 * time.Second)
	for {
		mu.Lock()
		got := len(chunks) > 0
		mu.Unlock()
		if got {
			break
		}
		select {
		case <-deadline2:
			t.Fatal("timeout waiting for content after denial")
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Wait for idle state (prompt response arrives).
	deadline3 := time.After(5 * time.Second)
	for client.State() != StateIdle {
		select {
		case <-deadline3:
			t.Fatalf("timeout waiting for idle, state=%s", client.State())
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Verify state transitions: idle → processing → waiting_permission → processing → idle.
	mu.Lock()
	defer mu.Unlock()

	expected := []string{StateIdle, StateProcessing, StateWaitingPermission, StateProcessing, StateIdle}
	if len(stateChanges) < len(expected) {
		t.Fatalf("state changes = %v, want at least %v", stateChanges, expected)
	}
	// Check the last N state transitions match expected.
	tail := stateChanges[len(stateChanges)-len(expected):]
	for i, want := range expected {
		if tail[i] != want {
			t.Errorf("stateChanges[%d] = %s, want %s (full: %v)", i, tail[i], want, stateChanges)
		}
	}

	if chunks[0].Text != "Hello from mock agent" {
		t.Errorf("chunk text = %q", chunks[0].Text)
	}
}
