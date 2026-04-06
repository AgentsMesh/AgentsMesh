//go:build integration

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
			Agent:     tools.AgentField("claude-code"),
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
			AvailableAgents: []tools.AgentSummary{
				{Slug: "claude-code", Name: "Claude Code"},
			},
		},
	}, nil
}

func (m *mockFormatClient) ListRepositories(_ context.Context) ([]tools.Repository, error) {
	return []tools.Repository{
		{ID: 1, Name: "my-repo", ProviderType: "gitlab", DefaultBranch: "main", HttpCloneURL: "https://git.example.com/repo.git"},
	}, nil
}

func (m *mockFormatClient) GetBindings(_ context.Context, _ *tools.BindingStatus) ([]tools.Binding, error) {
	return []tools.Binding{
		{ID: 1, InitiatorPod: "pod-a", TargetPod: "pod-b", Status: tools.BindingStatusActive, GrantedScopes: []tools.BindingScope{tools.ScopePodRead}},
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
		CreatedAt:         "2026-02-19T08:00:00Z", UpdatedAt: "2026-02-20T15:00:00Z",
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
		Status: tools.BindingStatusPending, PendingScopes: []tools.BindingScope{tools.ScopePodRead},
	}, nil
}

func (m *mockFormatClient) AcceptBinding(_ context.Context, _ int) (*tools.Binding, error) {
	return &tools.Binding{
		ID: 10, InitiatorPod: "other-pod", TargetPod: "test-pod",
		Status: tools.BindingStatusActive, GrantedScopes: []tools.BindingScope{tools.ScopePodRead},
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

func (m *mockFormatClient) GetPodSnapshot(_ context.Context, _ string, _ int, _ bool, _ bool) (*tools.PodSnapshot, error) {
	return &tools.PodSnapshot{
		PodKey: "target-pod", Output: "$ ls\nfile.go", TotalLines: 50, HasMore: false,
	}, nil
}

func (m *mockFormatClient) SendPodInput(_ context.Context, _ string, _ string, _ []string) error {
	return nil
}

func (m *mockFormatClient) CreatePod(_ context.Context, _ *tools.PodCreateRequest) (*tools.PodCreateResponse, error) {
	return &tools.PodCreateResponse{PodKey: "new-pod", Status: "initializing"}, nil
}

func (m *mockFormatClient) PostComment(_ context.Context, _ string, _ string, _ *int64) (*tools.TicketComment, error) {
	return &tools.TicketComment{ID: 1, Content: "test comment"}, nil
}

func (m *mockFormatClient) ListLoops(_ context.Context, _, _ string, _, _ int) ([]tools.LoopSummary, error) {
	return nil, nil
}

func (m *mockFormatClient) TriggerLoop(_ context.Context, _ string, _ map[string]interface{}) (*tools.LoopTriggerResult, error) {
	return nil, nil
}

// --- Mock status provider ---

type mockStatusProvider struct{}

func (m *mockStatusProvider) GetPodStatus(podKey string) (agentStatus string, podStatus string, shellPid int, found bool) {
	if podKey == "target-pod" {
		return "executing", "running", 12345, true
	}
	return "", "", 0, false
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
