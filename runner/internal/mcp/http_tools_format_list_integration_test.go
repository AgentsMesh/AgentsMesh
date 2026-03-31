//go:build integration

package mcp

import (
	"testing"
)

// --- List tools: verify Markdown table format (not JSON) ---

func TestFormatIntegration_ListAvailablePods(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "list_available_pods", `{}`)

	assertContains(t, text, "| Pod Key |")
	assertContains(t, text, "|---------|")
	assertContains(t, text, "| pod-abc |")
	assertContains(t, text, "| running |")
	assertContains(t, text, "| alice |")
	assertNotContains(t, text, `"pod_key"`)
}

func TestFormatIntegration_ListRunners(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "list_runners", `{}`)

	assertContains(t, text, "Runner #1")
	assertContains(t, text, "Node: node-1")
	assertContains(t, text, "Status: online")
	assertContains(t, text, "Pods: 2/5")
	assertContains(t, text, "Claude Code (claude-code)")
	assertNotContains(t, text, `"node_id"`)
}

func TestFormatIntegration_ListRepositories(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "list_repositories", `{}`)

	assertContains(t, text, "| ID |")
	assertContains(t, text, "| 1 |")
	assertContains(t, text, "| my-repo |")
	assertContains(t, text, "| gitlab |")
	assertNotContains(t, text, `"provider_type"`)
}

func TestFormatIntegration_GetBindings(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_bindings", `{}`)

	assertContains(t, text, "| ID |")
	assertContains(t, text, "| 1 |")
	assertContains(t, text, "| pod-a |")
	assertContains(t, text, "| active |")
	assertContains(t, text, "pod:read")
	assertNotContains(t, text, `"initiator_pod"`)
}

func TestFormatIntegration_GetBoundPods(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_bound_pods", `{}`)

	if text != "Bound pods: pod-x, pod-y" {
		t.Errorf("expected comma-separated pods, got: %s", text)
	}
}

func TestFormatIntegration_SearchChannels(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "search_channels", `{}`)

	assertContains(t, text, "| ID |")
	assertContains(t, text, "| general |")
	assertContains(t, text, "| 3 |")
	assertNotContains(t, text, `"member_count"`)
}

func TestFormatIntegration_GetChannelMessages(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "get_channel_messages", `{"channel_id":1}`)

	assertContains(t, text, "2 messages:")
	assertContains(t, text, "[2026-02-20T10:30:00Z] pod-alpha: Hello")
	assertContains(t, text, "[2026-02-20T10:31:00Z] pod-beta: Hi there")
	assertNotContains(t, text, `"sender_pod"`)
}

func TestFormatIntegration_SearchTickets(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "search_tickets", `{}`)

	assertContains(t, text, "| Slug |")
	assertContains(t, text, "| AM-123 |")
	assertContains(t, text, "| in_progress |")
	assertNotContains(t, text, `"slug"`)
}
