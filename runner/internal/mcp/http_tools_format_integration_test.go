package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// --- Mock collaboration client that returns real data ---

type mockFormatClient struct{}

func (m *mockFormatClient) GetPodKey() string { return "test-pod" }

func (m *mockFormatClient) ListAvailablePods(_ context.Context) ([]tools.AvailablePod, error) {
	return []tools.AvailablePod{
		{
			PodKey:    "pod-abc",
			Status:    tools.PodStatusRunning,
			AgentType: tools.AgentTypeField("claude-code"),
			CreatedBy: &tools.PodCreator{Username: "alice"},
			Ticket:    &tools.PodTicket{Title: "Fix bug"},
		},
	}, nil
}

func (m *mockFormatClient) ListRunners(_ context.Context) ([]tools.RunnerSummary, error) {
	return []tools.RunnerSummary{
		{
			ID: 1, NodeID: "node-1", Status: "online",
			CurrentPods: 2, MaxConcurrentPods: 5,
			AvailableAgents: []tools.AgentTypeSummary{
				{ID: 10, Slug: "claude-code", Name: "Claude Code"},
			},
		},
	}, nil
}

func (m *mockFormatClient) ListRepositories(_ context.Context) ([]tools.Repository, error) {
	return []tools.Repository{
		{ID: 1, Name: "my-repo", ProviderType: "gitlab", DefaultBranch: "main", CloneURL: "https://git.example.com/repo.git"},
	}, nil
}

func (m *mockFormatClient) GetBindings(_ context.Context, _ *tools.BindingStatus) ([]tools.Binding, error) {
	return []tools.Binding{
		{ID: 1, InitiatorPod: "pod-a", TargetPod: "pod-b", Status: tools.BindingStatusActive, GrantedScopes: []tools.BindingScope{tools.ScopeTerminalRead}},
	}, nil
}

func (m *mockFormatClient) GetBoundPods(_ context.Context) ([]string, error) {
	return []string{"pod-x", "pod-y"}, nil
}

func (m *mockFormatClient) SearchChannels(_ context.Context, _ string, _ *int, _ *string, _ *bool, _, _ int) ([]tools.Channel, error) {
	return []tools.Channel{
		{ID: 1, Name: "general", MemberCount: 3, Description: "General chat"},
	}, nil
}

func (m *mockFormatClient) GetMessages(_ context.Context, _ int, _, _, _ *string, _ int) ([]tools.ChannelMessage, error) {
	return []tools.ChannelMessage{
		{ID: 1, SenderPod: "pod-alpha", Content: "Hello", CreatedAt: "2026-02-20T10:30:00Z", MessageType: "text"},
		{ID: 2, SenderPod: "pod-beta", Content: "Hi there", CreatedAt: "2026-02-20T10:31:00Z", MessageType: "text"},
	}, nil
}

func (m *mockFormatClient) SearchTickets(_ context.Context, _ *int, _ *tools.TicketStatus, _ *tools.TicketPriority, _ *int, _ *string, _ string, _, _ int) ([]tools.Ticket, error) {
	return []tools.Ticket{
		{Slug: "AM-123", Title: "Fix auth bug", Status: tools.TicketStatusInProgress, Priority: tools.TicketPriorityHigh},
	}, nil
}

func (m *mockFormatClient) GetTicket(_ context.Context, _ string, _ *int, _ *int) (*tools.Ticket, error) {
	return &tools.Ticket{
		Slug: "AM-123", Title: "Fix auth bug",
		Status: tools.TicketStatusInProgress, Priority: tools.TicketPriorityHigh,
		ReporterName:      "john",
		ContentTotalLines: 5,
		ContentOffset:     0,
		ContentLimit:      5,
		CreatedAt: "2026-02-19T08:00:00Z", UpdatedAt: "2026-02-20T15:00:00Z",
	}, nil
}

func (m *mockFormatClient) CreateTicket(_ context.Context, _ *int64, _, _ string, _ tools.TicketPriority, _ *string) (*tools.Ticket, error) {
	return &tools.Ticket{
		Slug: "AM-200", Title: "New ticket",
		Status: tools.TicketStatusTodo, Priority: tools.TicketPriorityMedium,
		CreatedAt: "2026-02-21T10:00:00Z",
	}, nil
}

func (m *mockFormatClient) UpdateTicket(_ context.Context, _ string, _, _ *string, _ *tools.TicketStatus, _ *tools.TicketPriority) (*tools.Ticket, error) {
	return &tools.Ticket{
		Slug: "AM-123", Title: "Fix auth bug (updated)",
		Status: tools.TicketStatusDone, Priority: tools.TicketPriorityHigh,
		CreatedAt: "2026-02-19T08:00:00Z", UpdatedAt: "2026-02-21T10:00:00Z",
	}, nil
}

func (m *mockFormatClient) RequestBinding(_ context.Context, _ string, _ []tools.BindingScope) (*tools.Binding, error) {
	return &tools.Binding{
		ID: 10, InitiatorPod: "test-pod", TargetPod: "other-pod",
		Status: tools.BindingStatusPending, PendingScopes: []tools.BindingScope{tools.ScopeTerminalRead},
	}, nil
}

func (m *mockFormatClient) AcceptBinding(_ context.Context, _ int) (*tools.Binding, error) {
	return &tools.Binding{
		ID: 10, InitiatorPod: "other-pod", TargetPod: "test-pod",
		Status: tools.BindingStatusActive, GrantedScopes: []tools.BindingScope{tools.ScopeTerminalRead},
	}, nil
}

func (m *mockFormatClient) RejectBinding(_ context.Context, _ int, _ string) (*tools.Binding, error) {
	return &tools.Binding{
		ID: 10, InitiatorPod: "other-pod", TargetPod: "test-pod",
		Status: tools.BindingStatusRejected,
	}, nil
}

func (m *mockFormatClient) UnbindPod(_ context.Context, _ string) error { return nil }

func (m *mockFormatClient) GetChannel(_ context.Context, _ int) (*tools.Channel, error) {
	return &tools.Channel{
		ID: 1, Name: "dev-chat", Description: "Dev discussion", MemberCount: 5,
		CreatedByPod: "pod-leader", CreatedAt: "2026-02-19T08:00:00Z",
	}, nil
}

func (m *mockFormatClient) CreateChannel(_ context.Context, _, _ string, _ *int, _ *string) (*tools.Channel, error) {
	return &tools.Channel{
		ID: 2, Name: "new-channel", MemberCount: 1, CreatedAt: "2026-02-21T10:00:00Z",
	}, nil
}

func (m *mockFormatClient) SendMessage(_ context.Context, _ int, _ string, _ tools.ChannelMessageType, _ []string, _ *int) (*tools.ChannelMessage, error) {
	return &tools.ChannelMessage{
		ID: 100, ChannelID: 1, SenderPod: "test-pod", Content: "Hello",
		MessageType: "text", CreatedAt: "2026-02-21T10:00:00Z",
	}, nil
}

func (m *mockFormatClient) GetMessages2(_ context.Context, _ int, _, _, _ *string, _ int) ([]tools.ChannelMessage, error) {
	return nil, nil // unused duplicate avoidance
}

func (m *mockFormatClient) GetDocument(_ context.Context, _ int) (string, error) {
	return "doc content", nil
}

func (m *mockFormatClient) UpdateDocument(_ context.Context, _ int, _ string) error { return nil }

func (m *mockFormatClient) ObserveTerminal(_ context.Context, _ string, _ int, _ bool, _ bool) (*tools.TerminalOutput, error) {
	return &tools.TerminalOutput{
		PodKey: "target-pod", Output: "$ ls\nfile.go", TotalLines: 50, HasMore: false,
	}, nil
}

func (m *mockFormatClient) SendTerminalText(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockFormatClient) SendTerminalKey(_ context.Context, _ string, _ []string) error {
	return nil
}

func (m *mockFormatClient) CreatePod(_ context.Context, _ *tools.PodCreateRequest) (*tools.PodCreateResponse, error) {
	return &tools.PodCreateResponse{PodKey: "new-pod", Status: "initializing"}, nil
}

func (m *mockFormatClient) PostComment(_ context.Context, _ string, _ string, _ *int64) (*tools.TicketComment, error) {
	return &tools.TicketComment{ID: 1, Content: "test comment"}, nil
}

// --- Helper: register pod with mock client ---

func setupServerWithMockClient(t *testing.T) *HTTPServer {
	t.Helper()
	server := NewHTTPServer(nil, 9090)
	// Register a pod first, then replace its client with the mock
	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")
	server.mu.Lock()
	server.pods["test-pod"].Client = &mockFormatClient{}
	server.mu.Unlock()
	return server
}

// callTool sends a tools/call request and returns the text from the first content block.
func callTool(t *testing.T, server *HTTPServer, toolName string, args string) string {
	t.Helper()
	bodyStr := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + toolName + `","arguments":` + args + `}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(bodyStr))
	req.Header.Set("X-Pod-Key", "test-pod")
	rec := httptest.NewRecorder()

	server.handleMCP(rec, req)

	var resp MCPResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %d %s", resp.Error.Code, resp.Error.Message)
	}

	// Extract text from MCPToolResult
	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected result type: %T", resp.Result)
	}

	content, ok := resultMap["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("no content in result")
	}

	first, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("unexpected content type")
	}

	text, ok := first["text"].(string)
	if !ok {
		t.Fatal("no text in content")
	}

	return text
}

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
	assertContains(t, text, "[10] Claude Code (claude-code)")
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
	assertContains(t, text, "terminal:read")
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
	text := callTool(t, server, "bind_pod", `{"target_pod":"other-pod","scopes":["terminal:read"]}`)

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

func TestFormatIntegration_ObserveTerminal(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "observe_terminal", `{"pod_key":"target-pod"}`)

	assertContains(t, text, "Pod: target-pod")
	assertContains(t, text, "Lines: 50")
	assertContains(t, text, "$ ls")
	assertNotContains(t, text, `"total_lines"`)
}

func TestFormatIntegration_CreatePod(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "create_pod", `{"runner_id":1,"agent_type_id":10}`)

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

func TestFormatIntegration_SendTerminalText(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "send_terminal_text", `{"pod_key":"target-pod","text":"hello"}`)

	if text != "Text sent successfully" {
		t.Errorf("expected plain text, got: %s", text)
	}
}

func TestFormatIntegration_SendTerminalKey(t *testing.T) {
	server := setupServerWithMockClient(t)
	text := callTool(t, server, "send_terminal_key", `{"pod_key":"target-pod","keys":["enter"]}`)

	if text != "Keys sent successfully" {
		t.Errorf("expected plain text, got: %s", text)
	}
}

// --- Mock status provider ---

type mockStatusProvider struct{}

func (m *mockStatusProvider) GetPodStatus(podKey string) (agentStatus string, podStatus string, shellPid int, found bool) {
	if podKey == "target-pod" {
		return "executing", "running", 12345, true
	}
	return "", "", 0, false
}

// --- Assertion helpers ---

func assertContains(t *testing.T, text, substr string) {
	t.Helper()
	if !strings.Contains(text, substr) {
		t.Errorf("expected text to contain %q\ngot:\n%s", substr, text)
	}
}

func assertNotContains(t *testing.T, text, substr string) {
	t.Helper()
	if strings.Contains(text, substr) {
		t.Errorf("expected text NOT to contain %q\ngot:\n%s", substr, text)
	}
}
