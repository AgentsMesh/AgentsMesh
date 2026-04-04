package acp

import (
	"testing"
)

// newTestClient creates an ACPClient suitable for pure-logic unit tests.
// It does not start a subprocess.
func newTestClient() *ACPClient {
	return NewClient(ClientConfig{})
}

// --- AddPendingPermission / RemovePendingPermission ---

func TestACPClient_AddPendingPermission(t *testing.T) {
	c := newTestClient()

	c.AddPendingPermission(PermissionRequest{RequestID: "r1", ToolName: "exec"})
	c.AddPendingPermission(PermissionRequest{RequestID: "r2", ToolName: "write"})

	snap := c.GetSessionSnapshot()
	if len(snap.PendingPermissions) != 2 {
		t.Fatalf("expected 2 pending permissions, got %d", len(snap.PendingPermissions))
	}
	if snap.PendingPermissions[0].RequestID != "r1" {
		t.Errorf("PendingPermissions[0].RequestID = %q, want %q",
			snap.PendingPermissions[0].RequestID, "r1")
	}
	if snap.PendingPermissions[1].RequestID != "r2" {
		t.Errorf("PendingPermissions[1].RequestID = %q, want %q",
			snap.PendingPermissions[1].RequestID, "r2")
	}
}

func TestACPClient_RemovePendingPermission(t *testing.T) {
	c := newTestClient()

	c.AddPendingPermission(PermissionRequest{RequestID: "r1"})
	c.AddPendingPermission(PermissionRequest{RequestID: "r2"})
	c.AddPendingPermission(PermissionRequest{RequestID: "r3"})

	c.RemovePendingPermission("r2")

	snap := c.GetSessionSnapshot()
	if len(snap.PendingPermissions) != 2 {
		t.Fatalf("expected 2 pending permissions after removal, got %d",
			len(snap.PendingPermissions))
	}
	for _, p := range snap.PendingPermissions {
		if p.RequestID == "r2" {
			t.Error("r2 should have been removed")
		}
	}
}

func TestACPClient_RemovePendingPermission_NonExistent(t *testing.T) {
	c := newTestClient()

	c.AddPendingPermission(PermissionRequest{RequestID: "r1"})

	// Removing a non-existent ID should be a no-op.
	c.RemovePendingPermission("nonexistent")

	snap := c.GetSessionSnapshot()
	if len(snap.PendingPermissions) != 1 {
		t.Fatalf("expected 1 pending permission, got %d", len(snap.PendingPermissions))
	}
}

func TestACPClient_RemovePendingPermission_AllRemoved(t *testing.T) {
	c := newTestClient()

	c.AddPendingPermission(PermissionRequest{RequestID: "r1"})
	c.RemovePendingPermission("r1")

	snap := c.GetSessionSnapshot()
	if len(snap.PendingPermissions) != 0 {
		t.Fatalf("expected 0 pending permissions, got %d", len(snap.PendingPermissions))
	}
}

// --- GetSessionSnapshot ---

func TestACPClient_GetSessionSnapshot_Empty(t *testing.T) {
	c := newTestClient()

	snap := c.GetSessionSnapshot()
	if snap.State != StateUninitialized {
		t.Errorf("State = %q, want %q", snap.State, StateUninitialized)
	}
	if snap.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", snap.SessionID)
	}
	if len(snap.Messages) != 0 {
		t.Errorf("Messages should be empty, got %d", len(snap.Messages))
	}
	if len(snap.PendingPermissions) != 0 {
		t.Errorf("PendingPermissions should be empty, got %d", len(snap.PendingPermissions))
	}
}

func TestACPClient_GetSessionSnapshot_WithMessages(t *testing.T) {
	c := newTestClient()

	c.addMessage(ContentChunk{Text: "Hello", Role: "assistant"})
	c.addMessage(ContentChunk{Text: "World", Role: "user"})

	snap := c.GetSessionSnapshot()
	if len(snap.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(snap.Messages))
	}
	if snap.Messages[0].Text != "Hello" {
		t.Errorf("Messages[0].Text = %q, want %q", snap.Messages[0].Text, "Hello")
	}
	if snap.Messages[1].Role != "user" {
		t.Errorf("Messages[1].Role = %q, want %q", snap.Messages[1].Role, "user")
	}
}

func TestACPClient_GetSessionSnapshot_WithPermissions(t *testing.T) {
	c := newTestClient()

	c.AddPendingPermission(PermissionRequest{
		RequestID: "perm-1",
		ToolName:  "write_file",
	})
	c.addMessage(ContentChunk{Text: "Working...", Role: "assistant"})

	snap := c.GetSessionSnapshot()
	if len(snap.PendingPermissions) != 1 {
		t.Fatalf("expected 1 pending permission, got %d", len(snap.PendingPermissions))
	}
	if len(snap.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(snap.Messages))
	}
}

func TestACPClient_GetSessionSnapshot_IsolatedCopy(t *testing.T) {
	c := newTestClient()

	c.addMessage(ContentChunk{Text: "original", Role: "assistant"})
	snap := c.GetSessionSnapshot()

	// Mutating the snapshot should not affect the client
	snap.Messages[0].Text = "mutated"

	snap2 := c.GetSessionSnapshot()
	if snap2.Messages[0].Text != "original" {
		t.Errorf("snapshot mutation leaked: got %q, want %q",
			snap2.Messages[0].Text, "original")
	}
}
