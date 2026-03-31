package acp

import (
	"sync"
	"testing"
	"time"
)

// TestACPClient_PermissionRequest_EndToEnd verifies the full permission flow:
// Agent sends session/request_permission → Client receives OnPermissionRequest →
// Client calls RespondToPermission → Agent receives response → Agent continues.
func TestACPClient_PermissionRequest_EndToEnd(t *testing.T) {
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

	// Wait for the permission request to arrive.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		gotPerm := len(permReqs) > 0
		mu.Unlock()
		if gotPerm {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for permission request")
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Verify the permission request.
	mu.Lock()
	req := permReqs[0]
	mu.Unlock()
	if req.ToolName != "exec_command" {
		t.Errorf("ToolName = %q, want exec_command", req.ToolName)
	}
	// RequestID should be numeric (JSON-RPC id as string).
	if req.RequestID == "" {
		t.Error("RequestID is empty")
	}

	// State should be waiting_permission.
	if client.State() != StateWaitingPermission {
		t.Errorf("state = %s, want %s", client.State(), StateWaitingPermission)
	}

	// Approve the permission.
	if err := client.RespondToPermission(req.RequestID, true); err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	// Wait for content to arrive (agent sends content after permission is granted).
	deadline2 := time.After(5 * time.Second)
	for {
		mu.Lock()
		gotChunk := len(chunks) > 0
		mu.Unlock()
		if gotChunk {
			break
		}
		select {
		case <-deadline2:
			mu.Lock()
			t.Fatalf("timeout waiting for content, chunks=%d states=%v", len(chunks), stateChanges)
			mu.Unlock()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Verify content was received after permission approval.
	mu.Lock()
	defer mu.Unlock()
	if chunks[0].Text != "Hello from mock agent" {
		t.Errorf("chunk text = %q", chunks[0].Text)
	}
}

// TestACPClient_PermissionRequest_Snapshot verifies that pending permissions
// appear in the session snapshot.
func TestACPClient_PermissionRequest_Snapshot(t *testing.T) {
	var mu sync.Mutex
	var permReqs []PermissionRequest

	client := startMockClientWithMode(t, mockModeSendPerm, EventCallbacks{
		OnPermissionRequest: func(req PermissionRequest) {
			mu.Lock()
			permReqs = append(permReqs, req)
			mu.Unlock()
		},
	})
	defer client.Stop()

	if err := client.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	// Manually add a pending permission (simulates the flow in message_handler_acp.go).
	client.AddPendingPermission(PermissionRequest{
		RequestID: "42", ToolName: "exec", Description: "test",
	})

	snapshot := client.GetSessionSnapshot()
	if len(snapshot.PendingPermissions) != 1 {
		t.Fatalf("expected 1 pending permission, got %d", len(snapshot.PendingPermissions))
	}
	if snapshot.PendingPermissions[0].RequestID != "42" {
		t.Errorf("RequestID = %q", snapshot.PendingPermissions[0].RequestID)
	}
}
