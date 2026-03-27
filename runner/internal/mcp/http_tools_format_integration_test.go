//go:build integration

package mcp

import (
	"testing"
)

// --- Single entity tools: verify key-value text format (not JSON) ---

func TestFormatIntegration_GetTicket(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_ticket", `{"ticket_slug":"AM-123"}`)

	assertContains(t, text, "Ticket: AM-123 - Fix auth bug")
	assertContains(t, text, "Status: in_progress | Priority: high")
	assertContains(t, text, "Reporter: john")
	assertNotContains(t, text, "Description:")
	assertNotContains(t, text, `"identifier"`)
}

func TestFormatIntegration_GetTicketWithContentPagination(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_ticket", `{"ticket_slug":"AM-123","content_offset":0,"content_limit":100}`)

	assertContains(t, text, "Ticket: AM-123 - Fix auth bug")
	assertContains(t, text, "Status: in_progress | Priority: high")
	assertContains(t, text, "Reporter: john")
}

func TestFormatIntegration_CreateTicket(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "create_ticket", `{"title":"New ticket"}`)

	assertContains(t, text, "Ticket: AM-200 - New ticket")
	assertContains(t, text, "Status: todo")
}

func TestFormatIntegration_UpdateTicket(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "update_ticket", `{"ticket_slug":"AM-123","status":"done"}`)

	assertContains(t, text, "Ticket: AM-123 - Fix auth bug (updated)")
	assertContains(t, text, "Status: done")
}

func TestFormatIntegration_BindPod(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "bind_pod", `{"target_pod":"other-pod","scopes":["pod:read"]}`)

	assertContains(t, text, "Binding: #10")
	assertContains(t, text, "Initiator: test-pod")
	assertContains(t, text, "Status: pending")
	assertNotContains(t, text, `"initiator_pod"`)
}

func TestFormatIntegration_AcceptBinding(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "accept_binding", `{"binding_id":10}`)

	assertContains(t, text, "Binding: #10")
	assertContains(t, text, "Status: active")
}

func TestFormatIntegration_RejectBinding(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "reject_binding", `{"binding_id":10}`)

	assertContains(t, text, "Binding: #10")
	assertContains(t, text, "Status: rejected")
}

func TestFormatIntegration_GetChannel(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_channel", `{"channel_id":1}`)

	assertContains(t, text, "Channel: dev-chat (ID: 1)")
	assertContains(t, text, "Description: Dev discussion")
	assertContains(t, text, "Members: 5")
	assertNotContains(t, text, `"member_count"`)
}

func TestFormatIntegration_CreateChannel(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "create_channel", `{"name":"new-channel"}`)

	assertContains(t, text, "Channel: new-channel (ID: 2)")
}

func TestFormatIntegration_SendChannelMessage(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "send_channel_message", `{"channel_id":1,"content":"Hello"}`)

	assertContains(t, text, "Message #100")
	assertContains(t, text, "From: test-pod")
	assertContains(t, text, "Content: Hello")
}

func TestFormatIntegration_GetPodSnapshot(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_pod_snapshot", `{"pod_key":"target-pod"}`)

	assertContains(t, text, "Pod: target-pod")
	assertContains(t, text, "Lines: 50")
	assertContains(t, text, "$ ls")
	assertNotContains(t, text, `"total_lines"`)
}

func TestFormatIntegration_CreatePod(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "create_pod", `{"runner_id":1,"agent_slug":10}`)

	assertContains(t, text, "Pod: new-pod")
	assertContains(t, text, "Status: initializing")
	assertContains(t, text, "Binding: #10")
	assertNotContains(t, text, `"pod_key"`)
}

func TestFormatIntegration_GetPodStatus(t *testing.T) {
	server := setupServerWithMockClient(t)
	server.SetStatusProvider(&mockStatusProvider{})
	text := callTool(t, server, "get_pod_status", `{"pod_key":"target-pod"}`)

	assertContains(t, text, "Pod: target-pod")
	assertContains(t, text, "Agent: executing")
	assertContains(t, text, "Status: running")
	assertNotContains(t, text, `"agent_status"`)
}

// --- Existing text-returning tools remain unchanged ---

func TestFormatIntegration_UnbindPod(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "unbind_pod", `{"target_pod":"other-pod"}`)

	if text != "Pod unbound successfully" {
		t.Errorf("expected plain text, got: %s", text)
	}
}

func TestFormatIntegration_GetChannelDocument(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_channel_document", `{"channel_id":1}`)

	if text != "doc content" {
		t.Errorf("expected raw doc content, got: %s", text)
	}
}

func TestFormatIntegration_UpdateChannelDocument(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "update_channel_document", `{"channel_id":1,"document":"new content"}`)

	if text != "Document updated successfully" {
		t.Errorf("expected plain text, got: %s", text)
	}
}

func TestFormatIntegration_SendPodInput(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "send_pod_input", `{"pod_key":"target-pod","text":"hello","keys":["enter"]}`)

	if text != "Input sent successfully" {
		t.Errorf("expected plain text, got: %s", text)
	}
}
